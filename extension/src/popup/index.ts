/**
 * Popup script for the extension
 */

import { ExtensionConfig } from '../shared/types';
import { loadConfig, saveConfig } from '../shared/utils';

// DOM elements - will be initialized when DOM is ready
let enabledToggle: HTMLInputElement;
let enabledToggler: HTMLInputElement;
let apiUrlInput: HTMLInputElement;
let tokenInput: HTMLInputElement;
let deviceIdInput: HTMLInputElement;
let saveButton: HTMLButtonElement;
let testApiButton: HTMLButtonElement;
let apiTestResult: HTMLDivElement;
let statusText: HTMLSpanElement;
let versionText: HTMLSpanElement;
let isLocked: boolean;

/**
 * Load and display configuration
 */
const loadAndDisplayConfig = async (): Promise<void> => {
  try {
    const config = await loadConfig();

    // Update UI with config values
    enabledToggle.checked = config.enabled;
    apiUrlInput.value = config.api;
    deviceIdInput.value = config.id;
    tokenInput.value = config.token;
    isLocked = config.locked;

    // lock edit fields in extension locked mode
    if (isLocked) {
      saveButton.classList.add('hidden');
      enabledToggler.classList.add('hidden');
    } else {
      saveButton.classList.remove('hidden');
      enabledToggler.classList.remove('hidden');
    }
    enabledToggle.disabled = isLocked;
    apiUrlInput.disabled = isLocked;
    deviceIdInput.disabled = isLocked;
    tokenInput.disabled = isLocked;

    // Update status text
    statusText.textContent = config.enabled ? 'Active' : 'Disabled';
    statusText.style.color = config.enabled ? '#4CAF50' : '#F44336';

    // Get extension version
    const manifest = chrome.runtime.getManifest();
    versionText.textContent = manifest.version;
  } catch (error) {
    console.error('Error loading configuration:', error);
  }
};

/**
 * Validate URL format
 */
const isValidUrl = (url: string): boolean => {
  try {
    new URL(url);
    return true;
  } catch (e) {
    return false;
  }
};

/**
 * Save configuration
 */
const saveConfiguration = async (): Promise<void> => {
  try {
    // Validate API URL
    const apiUrl = apiUrlInput.value.trim();
    if (!isValidUrl(apiUrl)) {
      apiUrlInput.style.borderColor = '#F44336';
      saveButton.textContent = 'Invalid URL!';
      setTimeout(() => {
        saveButton.textContent = 'Save Settings';
        apiUrlInput.style.borderColor = '#ddd';
      }, 2000);
      return;
    }

    const token = tokenInput.value.trim();

    // Get current config
    const currentConfig = await loadConfig();

    // Create updated config
    if (! isLocked) {
      const updatedConfig: ExtensionConfig = {
        ...currentConfig,
        enabled: enabledToggle.checked,
        api: apiUrl,
        token: token,
      };

      // Save the updated config
      await saveConfig(updatedConfig);

      // Update status
      statusText.textContent = updatedConfig.enabled ? 'Active' : 'Disabled';
      statusText.style.color = updatedConfig.enabled ? '#4CAF50' : '#F44336';
    }

    // Show success message
    saveButton.textContent = 'Saved!';
    setTimeout(() => {
      saveButton.textContent = 'Save Settings';
    }, 2000);
  } catch (error) {
    console.error('Error saving configuration:', error);
    saveButton.textContent = 'Error!';
    setTimeout(() => {
      saveButton.textContent = 'Save Settings';
    }, 2000);
  }
};

/**
 * Test the API endpoint connection
 */
const testApiEndpoint = async (): Promise<void> => {
  const apiUrl = apiUrlInput.value.trim();
  const token = tokenInput.value.trim();

  // Validate URL format first
  if (!isValidUrl(apiUrl)) {
    showTestResult('Invalid URL format', false);
    return;
  }

  // Show testing status
  testApiButton.textContent = 'Testing...';
  testApiButton.disabled = true;

  try {
    // Try to connect to the health endpoint
    const healthEndpoint = `${apiUrl}/api/health`;

    // Create an AbortController for timeout
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 5000);

    const headers = new Headers();
    if (token != "") {
      headers.set('Authorization', 'Bearer ' + token);
    }

    const response = await fetch(healthEndpoint, { 
      method: 'GET',
      signal: controller.signal,
      headers: headers,
    });

    // Clear the timeout
    clearTimeout(timeoutId);

    if (response.ok) {
      showTestResult('Connection successful!', true);
    } else {
      showTestResult(`Connection failed: ${response.status} ${response.statusText}`, false);
    }
  } catch (error) {
    showTestResult(`Connection failed: ${error instanceof Error ? error.message : String(error)}`, false);
  } finally {
    // Reset button state
    testApiButton.textContent = 'Test';
    testApiButton.disabled = false;
  }
};

/**
 * Show test result with appropriate styling
 */
const showTestResult = (message: string, success: boolean): void => {
  apiTestResult.textContent = message;
  apiTestResult.style.color = success ? '#4CAF50' : '#F44336';
  apiTestResult.style.display = 'block';

  // Hide the message after 5 seconds
  setTimeout(() => {
    apiTestResult.style.display = 'none';
  }, 5000);
};

/**
 * Initialize the popup
 */
const initialize = (): void => {
  // Initialize DOM elements
  enabledToggle = document.getElementById('enabled-toggle') as HTMLInputElement;
  enabledToggler = document.getElementById('enabled-toggler') as HTMLInputElement;
  apiUrlInput = document.getElementById('api-url') as HTMLInputElement;
  tokenInput = document.getElementById('token') as HTMLInputElement;
  deviceIdInput = document.getElementById('device-id') as HTMLInputElement;
  saveButton = document.getElementById('save-button') as HTMLButtonElement;
  testApiButton = document.getElementById('test-api-button') as HTMLButtonElement;
  apiTestResult = document.getElementById('api-test-result') as HTMLDivElement;
  statusText = document.getElementById('status-text') as HTMLSpanElement;
  versionText = document.getElementById('version-text') as HTMLSpanElement;

  // Check if all elements were found
  if (!enabledToggle || !enabledToggler || !apiUrlInput || !tokenInput || !deviceIdInput || 
      !saveButton || !testApiButton || !apiTestResult || !statusText || !versionText) {
    console.error('Some DOM elements were not found');
    return;
  }

  // Load and display configuration
  loadAndDisplayConfig();

  // Set up event listeners
  console.log(saveButton);
  console.log(testApiButton);
  console.log(enabledToggle);

  saveButton.addEventListener('click', saveConfiguration);
  testApiButton.addEventListener('click', testApiEndpoint);
  
  // Toggle status text when the toggle is clicked
  enabledToggle.addEventListener('change', () => {
    statusText.textContent = enabledToggle.checked ? 'Active' : 'Disabled';
    statusText.style.color = enabledToggle.checked ? '#4CAF50' : '#F44336';
  });
};

// Initialize when the DOM is ready
// Use a more reliable approach for Chrome extensions
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initialize);
} else {
  // DOM is already ready
  initialize();
}
