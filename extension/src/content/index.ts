/**
 * Content script for detecting login events
 */

import { LoginData, Message, MessageType } from '../shared/types';
import { getDomain } from '../shared/utils';

// Store form data temporarily
let formData: { [key: string]: string } = {};
let passwordFields: HTMLInputElement[] = [];
let usernameFields: HTMLInputElement[] = [];

/**
 * Determine if an input field is likely a username field
 */
const isUsernameField = (input: HTMLInputElement): boolean => {
  const type = input.type.toLowerCase();
  const name = input.name.toLowerCase();
  const id = input.id.toLowerCase();
  const placeholder = (input.placeholder || '').toLowerCase();

  // Check for email type
  if (type === 'email') return true;

  // Check for common username field names
  const usernamePatterns = [
    'user', 'username', 'email', 'login', 'account', 'id', 'identifier'
  ];

  return usernamePatterns.some(pattern => 
    name.includes(pattern) || id.includes(pattern) || placeholder.includes(pattern)
  );
};

/**
 * Determine if an input field is a password field
 */
const isPasswordField = (input: HTMLInputElement): boolean => {
  return input.type.toLowerCase() === 'password';
};

// Track elements that already have event listeners
const monitoredElements = new WeakSet();

// Event handler functions
const passwordChangeHandler = (e: Event) => {
  formData['password'] = (e.target as HTMLInputElement).value;
};

const passwordInputHandler = (e: Event) => {
  formData['password'] = (e.target as HTMLInputElement).value;
};

const usernameChangeHandler = (e: Event) => {
  formData['username'] = (e.target as HTMLInputElement).value;
};

const usernameInputHandler = (e: Event) => {
  formData['username'] = (e.target as HTMLInputElement).value;
};

const buttonClickHandler = () => {

  // If we already have the data in formData, use it
  if (formData['username'] && formData['password']) {
    sendLoginData();
    return;
  }

  // Try to get values directly from the fields
  if (passwordFields.length > 0 && usernameFields.length > 0) {
    for (let i = 0; i < Math.min(passwordFields.length, usernameFields.length); i++) {
      formData['password'] = passwordFields[i].value;
      formData['username'] = usernameFields[i].value;

      if (formData['username'] && formData['password']) {
        sendLoginData();
        return;
      }
    }
  }

  // Last resort: try to find any password and username fields on the page
  const allPasswordFields = document.querySelectorAll('input[type="password"]');
  const allUsernameFields = Array.from(document.querySelectorAll('input')).filter(
    input => isUsernameField(input as HTMLInputElement)
  );

  if (allPasswordFields.length > 0 && allUsernameFields.length > 0) {
    formData['password'] = (allPasswordFields[0] as HTMLInputElement).value;
    formData['username'] = (allUsernameFields[0] as HTMLInputElement).value;

    if (formData['username'] && formData['password']) {
      sendLoginData();
    }
  }
};

/**
 * Find all forms on the page and attach event listeners
 */
const findAndMonitorForms = (): void => {
  // Reset stored data
  formData = {};
  passwordFields = [];
  usernameFields = [];

  // Find all forms
  const forms = document.querySelectorAll('form');

  forms.forEach((form, index) => {
    // Skip if this form already has our event listener
    if (monitoredElements.has(form)) {
      return;
    }

    // Find input fields
    const inputs = form.querySelectorAll('input');

    // Identify username and password fields
    inputs.forEach(input => {
      // Skip if this input already has our event listener
      if (monitoredElements.has(input)) {
        return;
      }

      if (isPasswordField(input as HTMLInputElement)) {
        passwordFields.push(input as HTMLInputElement);

        // Monitor password field changes
        input.addEventListener('change', passwordChangeHandler);
        input.addEventListener('input', passwordInputHandler);
        monitoredElements.add(input);
        //console.log('Added event listeners to password field');
      } else if (isUsernameField(input as HTMLInputElement)) {
        usernameFields.push(input as HTMLInputElement);

        // Monitor username field changes
        input.addEventListener('change', usernameChangeHandler);
        input.addEventListener('input', usernameInputHandler);
        monitoredElements.add(input);
      }
    });

    // Monitor form submission
    form.addEventListener('submit', handleFormSubmit);
    monitoredElements.add(form);
  });

  // Also monitor for button clicks that might trigger login
  const buttons = document.querySelectorAll('button, input[type="submit"], input[type="button"]');

  buttons.forEach((button, index) => {
    // Skip if this button already has our event listener
    if (monitoredElements.has(button)) {
      return;
    }

    button.addEventListener('click', buttonClickHandler);
    monitoredElements.add(button);
  });
};

/**
 * Handle form submission
 */
const handleFormSubmit = (event: Event): void => {

  // If we have both username and password, send the data
  if (formData['username'] && formData['password']) {
    sendLoginData();
    return;
  }

  // Try to get values directly from the fields
  if (passwordFields.length > 0 && usernameFields.length > 0)
  {
    for (let i = 0; i < Math.min(passwordFields.length, usernameFields.length); i++)
    {
      formData['password'] = passwordFields[i].value;
      formData['username'] = usernameFields[i].value;

      if (formData['username'] && formData['password']) {
        sendLoginData();
        return;
      }
    }

  }
};

// Track last sent login data to prevent duplicates
let lastSentLogin: {
  domain: string;
  username: string;
  timestamp: number;
} | null = null;

// Debounce time in milliseconds
const DEBOUNCE_TIME = 1000; // 1 second

/**
 * Send login data to the background script
 */
const sendLoginData = (): void => {
  const domain = getDomain(window.location.href);
  const username = formData['username'];
  const password = formData['password'];
  const currentTime = Date.now();

  // Check if this is a duplicate submission
  if (lastSentLogin && 
      lastSentLogin.domain === domain && 
      lastSentLogin.username === username && 
      (currentTime - lastSentLogin.timestamp) < DEBOUNCE_TIME) {
    return;
  }

  // Update last sent login data
  lastSentLogin = {
    domain,
    username,
    timestamp: currentTime
  };

  const loginData: Partial<LoginData> = {
    domain,
    username,
    password,
  };

  // Send message to background script
  chrome.runtime.sendMessage({
    type: MessageType.LOGIN_DETECTED,
    data: loginData
  });

  // Clear stored data for security
  formData = {};
};

/**
 * Monitor DOM changes to detect dynamically added forms
 */
const observeDOMChanges = (): void => {
  const observer = new MutationObserver((mutations) => {
    let shouldScan = false;

    mutations.forEach(mutation => {
      if (mutation.addedNodes.length > 0) {
        shouldScan = true;
      }
    });

    if (shouldScan) {
      findAndMonitorForms();
    }
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true
  });
};

/**
 * Initialize the content script
 */
const initialize = (): void => {
  //console.log('Content script initialized for:', window.location.href);

  // Initial scan for forms
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      //console.log('DOMContentLoaded fired, scanning for forms');
      findAndMonitorForms();
      observeDOMChanges();
    });
  } else {
    findAndMonitorForms();
    observeDOMChanges();
  }
};

// Start the content script
initialize();
