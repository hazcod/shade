/**
 * Represents login data captured by the extension
 */
export interface LoginData {
  domain: string;
  username: string;
  password: string;
  deviceId: string;
  capturedTime?: string;
}

/**
 * Response from the backend API
 */
export interface ApiResponse {
  status: 'success' | 'error';
  message?: string;
}

/**
 * Message types for communication between content script and background script
 */
export enum MessageType {
  LOGIN_DETECTED = 'LOGIN_DETECTED',
  GET_DEVICE_ID = 'GET_DEVICE_ID',
}

/**
 * Message structure for internal extension communication
 */
export interface Message {
  type: MessageType;
  data?: any;
}

/**
 * Configuration for the extension
 */
export interface ExtensionConfig {
  api: string;
  id: string;
  enabled: boolean;
  token: string;
  locked: boolean;
  filters: string[];
}

/**
 * Default configuration values
 */
export const DEFAULT_CONFIG: ExtensionConfig = {
  api: 'http://localhost:8080',
  id: '',
  enabled: true,
  token: '',
  locked: false,
  filters: [],
};