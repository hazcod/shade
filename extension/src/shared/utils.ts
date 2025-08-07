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
    let port = urlObj.port;
    if (port == '') {
      port = urlObj.protocol == 'https:' ? '443' : '80';
    }
    return urlObj.protocol + '//' + urlObj.hostname + ':' + port;
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
      const config: ExtensionConfig = result.config || { ...DEFAULT_CONFIG };

      // Normalize legacy field name if present
      // If a previous version stored `deviceId`, migrate it to `id`.
      const anyConfig: any = config as any;
      if (!config.id && anyConfig.deviceId) {
        config.id = anyConfig.deviceId;
        delete anyConfig.deviceId;
      }
      
      // Generate device ID if not present (store in `id` as per schema)
      if (!config.id) {
        config.id = generateDeviceId();
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

  // Load token from config to include Authorization header when available
  let token = '';
  try {
    const config = await loadConfig();
    token = config.token || '';
  } catch (e) {
    // Ignore config load error; proceed without auth header
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  return fetch(url, {
    method: 'POST',
    headers,
    body: JSON.stringify(data),
  });
};