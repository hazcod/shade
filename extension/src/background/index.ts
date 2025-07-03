/**
 * Background script for the extension
 */

import { LoginData, MessageType } from '../shared/types';
import { loadConfig, sendToBackend } from '../shared/utils';

async function sha(mode: string, input: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(input);
  const hashBuffer = await crypto.subtle.digest(mode, data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

const checkHIBP = async(
    partialLoginData: Partial<LoginData>,
): Promise<void> => {
  try {
    // HIBP needs a SHA1 hash
    let hashedPassword = (await sha('SHA-1', <string>partialLoginData.password)).toUpperCase();
    const hashPrefix = hashedPassword.substring(0, 5);
    const hashSuffix = hashedPassword.substring(5);

    // check HIBP using their privacy-preserving API
    let url = 'https://api.pwnedpasswords.com/range/' + hashPrefix;
    let response = await fetch(url, { method: 'GET' });
    const responseText = await response.text();

    if (!response.ok) {
      console.error('Failed to fetch HIBP:', responseText);
    }

    const lines = responseText.split('\n');
    const found = lines.some(line => line.startsWith(hashSuffix));

    // check the response output if it contains our hash
    if (! found) {
      return;
    }

    // the password is found in HIBP!
    console.log('Password for ' + partialLoginData.domain + ' was found on HIBP.');

    // Use a data URL for the icon to avoid SVG compatibility issues
    try {
      chrome.notifications.create({
        type: "basic",
        iconUrl: chrome.runtime.getURL('icons/warning.png'),
        title: "Password Warning!",
        message: "Your password for " + partialLoginData.domain + " has been leaked on the internet!",
        priority: 2,
      });
    } catch (error) {
      console.error('Failed to create HIBP notification:', error);
    }

  } catch (error) {
    console.error('failed to check HIBP', error);
  }
};

/**
 * Handle login detection
 */
const handleLoginDetected = async (
  partialLoginData: Partial<LoginData>,
  apiUrl: string,
  deviceId: string,
  filters: string[],
): Promise<void> => {
  try {
    // Complete the login data
    const loginData: LoginData = {
      domain: partialLoginData.domain || '',
      username: partialLoginData.username || '',
      password: partialLoginData.password || '',
      deviceId: deviceId,
      capturedTime: new Date().toISOString(),
    };

    if (Array.isArray(filters) && filters.length > 0) {
      let found = false;

      for (const filter of filters) {
        if (loginData.username.includes(filter)) {
          found = true;
          break;
        }
      }

      if (! found) {
        return;
      }
    }

    // Log the detection (excluding password for security in logs)
    console.log(`Login detected on ${loginData.domain} for user ${loginData.username}`);

    // only send the credentials when it's either localhost or a secure remote endpoint
    if (!apiUrl.startsWith("https://") && !apiUrl.includes("localhost")) {
      console.error('Refusing to send credentials to insecure endpoint: ', apiUrl);
      return;
    }

    // calculate the hash of this
    let hashedPassword = await sha('SHA-512', loginData.password);

    // send to backend
    const response = await sendToBackend('/api/login/register', {
      domain: loginData.domain,
      username: loginData.username,
      hash: hashedPassword,
      device_id: loginData.deviceId,
      captured_time: loginData.capturedTime,
    }, apiUrl);

    if (!response.ok) {
      const errorData = await response.json();
      console.error('Failed to send login data:', errorData);
    }
  } catch (error) {
    console.error('Error handling login detection:', error);
  }
};

/**
 * Initialize the background script
 */
const initialize = async (): Promise<void> => {
  // Load or initialize configuration
  await loadConfig();

  // Set up message listener using the recommended pattern for Manifest V3
  // This approach properly handles asynchronous responses in service workers
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    // Create a promise to handle the message asynchronously
    const responsePromise = (async () => {
      try {
        // Load configuration
        const config = await loadConfig();

        // Skip processing if extension is disabled
        if (!config.enabled) {
          return { success: false, error: 'Extension is disabled' };
        }

        switch (message.type) {
          case MessageType.LOGIN_DETECTED:
            await handleLoginDetected(message.data, config.api, config.id, config.filters);
            checkHIBP(message.data).then(r => {});
            return { success: true };

          case MessageType.GET_DEVICE_ID:
            return { deviceId: config.id };

          default:
            console.warn('Unknown message type:', message.type);
            return { success: false, error: 'Unknown message type' };
        }
      } catch (error) {
        console.error('Error handling message:', error);
        return { success: false, error: 'Internal error' };
      }
    })();

    // Send the response when the promise resolves
    // This is the key part of the fix - we chain the sendResponse to the promise resolution
    responsePromise.then(sendResponse);

    // Return true to indicate we will send a response asynchronously
    // This is required when using asynchronous sendResponse in Manifest V3
    return true;
  });

  console.log('Login Detector background script initialized');
};

// Start the background script
initialize();
