import {
  CheckDeviceAuth,
  CheckGoogleDriveAuth,
  ChooseExportDirectory,
  DisconnectGoogleDrive,
  ExchangeOAuthCode,
  ExportSettingsBackup,
  GetAppState,
  GetStandardAuthURL,
  ImportSettingsBackup,
  Logout,
  RunExport,
  SaveSchedule,
  SaveSettings,
  StartDeviceAuth,
  StartGoogleDriveAuth,
} from "../../wailsjs/go/main/App";
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
import type {
  AppState,
  DeviceAuthSession,
  DeviceAuthStatus,
  ExportRequest,
  ExportResult,
  GoogleDriveAuthSession,
  GoogleDriveAuthStatus,
  SaveScheduleInput,
  SaveSettingsInput,
} from "@/types/simkl";

export const getAppState = () => GetAppState() as Promise<AppState>;
export const saveSettings = (input: SaveSettingsInput) =>
  SaveSettings(input as unknown as Parameters<typeof SaveSettings>[0]) as Promise<AppState>;
export const saveSchedule = (input: SaveScheduleInput) =>
  SaveSchedule(input as unknown as Parameters<typeof SaveSchedule>[0]) as Promise<AppState>;
export const chooseExportDirectory = () =>
  ChooseExportDirectory() as Promise<string>;
export const selectDirectory = chooseExportDirectory;
export const startDeviceAuth = () =>
  StartDeviceAuth() as Promise<DeviceAuthSession>;
export const checkDeviceAuth = () =>
  CheckDeviceAuth() as Promise<DeviceAuthStatus>;
export const startGoogleDriveAuth = () =>
  StartGoogleDriveAuth() as Promise<GoogleDriveAuthSession>;
export const checkGoogleDriveAuth = () =>
  CheckGoogleDriveAuth() as Promise<GoogleDriveAuthStatus>;
export const disconnectGoogleDrive = () =>
  DisconnectGoogleDrive() as Promise<AppState>;
export const logout = () => Logout() as Promise<AppState>;
export const runExport = (request: ExportRequest) =>
  RunExport(request as unknown as Parameters<typeof RunExport>[0]) as Promise<ExportResult>;
export const getStandardAuthURL = () => GetStandardAuthURL();
export const exchangeOAuthCode = (code: string) =>
  ExchangeOAuthCode(code) as Promise<AppState>;
export const exportSettingsBackup = (password: string) =>
  ExportSettingsBackup(password) as Promise<string>;
export const importSettingsBackup = (password: string) =>
  ImportSettingsBackup(password) as Promise<string>;
export const openExternal = (url: string) => BrowserOpenURL(url);
