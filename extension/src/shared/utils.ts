/**
 * Utility functions for the extension
 */

import { ExtensionConfig, DEFAULT_CONFIG } from './types';

/**
 * Generate a unique device ID
 */
export const generateDeviceId = (): string => {
  return 'device_' + Math.random().toString(36).substring(2, 15) + 
         Math.random().toString(36).substring(2, 15);
};

/**
 * Get the current domain from a URL
 */
export const getDomain = (url: string): string => {
  try {
    const urlObj = new URL(url);
    return urlObj.hostname;
  } catch (e) {
    console.error('Invalid URL:', url);
    return '';
  }
};

/**
 * Load extension configuration from storage
 */
export const loadConfig = async (): Promise<ExtensionConfig> => {
  return new Promise((resolve) => {
    chrome.storage.local.get('config', (result) => {
      const config = result.config || DEFAULT_CONFIG;
      
      // Generate device ID if not present
      if (!config.deviceId) {
        config.deviceId = generateDeviceId();
        chrome.storage.local.set({ config });
      }
      
      resolve(config);
    });
  });
};

/**
 * Save extension configuration to storage
 */
export const saveConfig = async (config: ExtensionConfig): Promise<void> => {
  return new Promise((resolve) => {
    chrome.storage.local.set({ config }, resolve);
  });
};

/**
 * Send data to the backend API
 */
export const sendToBackend = async (
  endpoint: string, 
  data: any, 
  apiUrl: string
): Promise<Response> => {
  const url = `${apiUrl}${endpoint}`;
  
  return fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
};