/**
 * Content script for detecting login events
 */

import { LoginData, Message, MessageType } from '../shared/types';
import { getDomain } from '../shared/utils';

// Store form data temporarily
let formData: { [key: string]: string } = {};
let passwordFields: HTMLInputElement[] = [];
let usernameFields: HTMLInputElement[] = [];
let mfaFields: HTMLInputElement[] = [];
let hasMFADetected = false;
let detectedMFAType = '';

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

  // Check for model username field names
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

/**
 * Determine if an input field is likely an MFA/TOTP field
 */
const isMFAField = (input: HTMLInputElement): boolean => {
  const type = input.type.toLowerCase();
  const name = input.name.toLowerCase();
  const id = input.id.toLowerCase();
  const placeholder = (input.placeholder || '').toLowerCase();
  const className = input.className.toLowerCase();
  
  // Check for numeric input types commonly used for TOTP
  if (type === 'number' || type === 'tel') {
    // Check for TOTP/MFA related patterns
    const mfaPatterns = [
      'totp', 'mfa', 'otp', 'code', 'token', 'verification', 'verify', 
      'authenticator', 'auth', '2fa', 'twofactor', 'security', 'sms', 'mfa', 'multifactor'
    ];
    
    const hasPattern = mfaPatterns.some(pattern => 
      name.includes(pattern) || id.includes(pattern) || 
      placeholder.includes(pattern) || className.includes(pattern)
    );
    
    if (hasPattern) {
      detectedMFAType = 'TOTP';
      return true;
    }
  }
  
  // Check for text fields that might be used for codes
  if (type === 'text') {
    const mfaPatterns = [
      'totp', 'mfa', 'otp', 'code', 'token', 'verification', 'verify',
      'authenticator', 'auth', '2fa', 'twofactor', 'security', 'mfa', 'multifactor'
    ];
    
    const hasPattern = mfaPatterns.some(pattern => 
      name.includes(pattern) || id.includes(pattern) || 
      placeholder.includes(pattern) || className.includes(pattern)
    );
    
    // Also check for typical TOTP field characteristics
    const maxLength = input.maxLength;
    const autocomplete = (input.autocomplete || '').toLowerCase();
    const inputMode = (input.inputMode || '').toLowerCase();
    
    // Check for specific TOTP indicators
    const isTOTPField = hasPattern || 
                       (maxLength >= 4 && maxLength <= 8) ||
                       autocomplete === 'one-time-code' ||
                       inputMode === 'numeric';
    
    if (isTOTPField) {
      detectedMFAType = 'TOTP';
      return true;
    }
  }
  
  return false;
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
  mfaFields = [];
  hasMFADetected = false;
  detectedMFAType = '';

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
      } else if (isMFAField(input as HTMLInputElement)) {
        mfaFields.push(input as HTMLInputElement);
        hasMFADetected = true;
        
        // If we have pending login data and just detected MFA, monitor for success before sending
        if (pendingLoginData && pendingLoginData.timeoutId) {
          clearTimeout(pendingLoginData.timeoutId);
          monitorLoginResult(
            pendingLoginData.domain,
            pendingLoginData.username,
            pendingLoginData.password,
            true,
            detectedMFAType
          );
          lastSentLogin = {
            domain: pendingLoginData.domain,
            username: pendingLoginData.username,
            timestamp: pendingLoginData.timestamp
          };
          pendingLoginData = null;
        }
        
        // Monitor MFA field changes
        input.addEventListener('change', () => {
          formData['mfa'] = (input as HTMLInputElement).value;
        });
        input.addEventListener('input', () => {
          formData['mfa'] = (input as HTMLInputElement).value;
        });
        monitoredElements.add(input);
        //console.log('MFA field detected:', detectedMFAType);
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

// Track pending login data for multi-step authentication
let pendingLoginData: {
  domain: string;
  username: string;
  password: string;
  timestamp: number;
  timeoutId?: number;
} | null = null;

// Wait time for potential MFA step (in milliseconds)
const MFA_WAIT_TIME = 8000; // 8 seconds

// Wait time for login success detection (in milliseconds)
const LOGIN_SUCCESS_WAIT_TIME = 5000; // 5 seconds

/**
 * Check if the current page indicates a login failure
 */
const isLoginFailure = (): boolean => {
  // Check for common error message patterns
  const errorSelectors = [
    '[class*="error"]',
    '[class*="invalid"]',
    '[class*="fail"]',
    '[id*="error"]',
    '[id*="invalid"]',
    '[id*="fail"]',
    '.alert-danger',
    '.alert-error',
    '.error-message',
    '.login-error',
    '.auth-error'
  ];

  for (const selector of errorSelectors) {
    const elements = document.querySelectorAll(selector);
    for (const element of Array.from(elements)) {
      const text = element.textContent?.toLowerCase() || '';
      const errorPatterns = [
        'invalid', 'incorrect', 'wrong', 'failed', 'error', 'denied',
        'unauthorized', 'authentication failed', 'login failed',
        'bad credentials', 'invalid username', 'invalid password',
        'account locked', 'too many attempts'
      ];
      
      if (errorPatterns.some(pattern => text.includes(pattern))) {
        return true;
      }
    }
  }

  // Check for specific error input styling
  const inputs = document.querySelectorAll('input[type="password"], input[type="email"], input[type="text"]');
  for (const input of Array.from(inputs)) {
    if (input.classList.contains('error') || input.classList.contains('invalid') || 
        input.classList.contains('is-invalid') || input.getAttribute('aria-invalid') === 'true') {
      return true;
    }
  }

  return false;
};

/**
 * Check if the current page indicates a login success
 */
const isLoginSuccess = (initialUrl?: string): boolean => {
  const currentUrl = window.location.href;
  const currentPath = window.location.pathname;

  // Check for success elements on the page
  const successSelectors = [
    '[class*="welcome"]',
    '[class*="dashboard"]',
    '[class*="success"]',
    '[id*="welcome"]',
    '[id*="dashboard"]',
    '[id*="success"]',
    '.alert-success',
    '.success-message',
    '.login-success'
  ];

  for (const selector of successSelectors) {
    const elements = document.querySelectorAll(selector);
    if (elements.length > 0) {
      return true;
    }
  }

  // Check for logout buttons or user menus (indicates successful login)
  const loggedInIndicators = [
    'a[href*="logout"]',
    'button[onclick*="logout"]',
    '[class*="user-menu"]',
    '[class*="profile-menu"]',
    '[class*="account-menu"]'
  ];

  for (const selector of loggedInIndicators) {
    if (document.querySelector(selector)) {
      return true;
    }
  }

  return false;
};

/**
 * Monitor for login success or failure after form submission
 */
const monitorLoginResult = (domain: string, username: string, password: string, hasMFA: boolean, mfaType?: string): void => {
  const startTime = Date.now();
  const initialUrl = window.location.href;
  
  const checkResult = () => {
    const elapsed = Date.now() - startTime;
    
    // Check if login failed
    if (isLoginFailure()) {
      console.log('Login failure detected, not sending data');
      return; // Don't send data for failed logins
    }
    
    // Check if login succeeded
    if (isLoginSuccess(initialUrl) || window.location.href !== initialUrl) {
      console.log('Login success detected, sending data');
      sendLoginDataImmediate(domain, username, password, hasMFA, mfaType);
      return;
    }
    
    // Continue monitoring if time hasn't elapsed
    if (elapsed < LOGIN_SUCCESS_WAIT_TIME) {
      setTimeout(checkResult, 500); // Check every 500ms
    } else {
      // Timeout reached - assume success if no clear failure detected
      console.log('Login result timeout, assuming success');
      sendLoginDataImmediate(domain, username, password, hasMFA, mfaType);
    }
  };
  
  // Start monitoring after a brief delay to allow page to update
  setTimeout(checkResult, 1000);
};

/**
 * Send login data immediately to the background script
 */
const sendLoginDataImmediate = (domain: string, username: string, password: string, hasMFA: boolean, mfaType?: string): void => {
  const loginData: Partial<LoginData> = {
    domain,
    username,
    password,
    hasMFA,
    mfaType,
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
 * Send login data with delay to check for MFA
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

  // If MFA is already detected on current page, monitor for success before sending
  if (hasMFADetected) {
    monitorLoginResult(domain, username, password, true, detectedMFAType);
    lastSentLogin = { domain, username, timestamp: currentTime };
    return;
  }

  // Cancel any existing pending login
  if (pendingLoginData && pendingLoginData.timeoutId) {
    clearTimeout(pendingLoginData.timeoutId);
  }

  // Set up delayed sending to wait for potential MFA
  pendingLoginData = {
    domain,
    username,
    password,
    timestamp: currentTime,
    timeoutId: window.setTimeout(() => {
      // After waiting, check if MFA was detected
      const finalHasMFA = hasMFADetected;
      const finalMFAType = finalHasMFA ? detectedMFAType : undefined;
      
      // Monitor for login success before sending data
      monitorLoginResult(domain, username, password, finalHasMFA, finalMFAType);
      
      lastSentLogin = { domain, username, timestamp: currentTime };
      pendingLoginData = null;
    }, MFA_WAIT_TIME)
  };
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
