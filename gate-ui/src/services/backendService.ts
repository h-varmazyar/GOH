import { api } from './api';

export type ConfigValue = string | number | boolean | null | Record<string, unknown> | unknown[];

export interface BackendConfig {
  key: string;
  value: ConfigValue;
  description?: string;
  type?: string;
  editable: boolean;
}

export interface Worker {
  id: string;
  name: string;
  section: string;
  enabled: boolean;
  isActive: boolean;
  status: string;
}

export type WorkerSetting = BackendConfig;

const asRecord = (value: unknown): Record<string, unknown> =>
  value && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : {};

const parseResponse = (response: unknown): unknown => {
  if (typeof response !== 'string') return response;
  try {
    return JSON.parse(response) as unknown;
  } catch {
    return response;
  }
};

const getSectionItems = (response: unknown, section: string): unknown[] => {
  const parsed = parseResponse(response);
  if (Array.isArray(parsed)) return parsed;

  const root = asRecord(parsed);
  const containers = [root, asRecord(root.data), asRecord(root.Data), asRecord(root.sections), asRecord(root.Sections)];
  for (const container of containers) {
    const items = container[section] ?? container.items ?? container.Items;
    if (Array.isArray(items)) return items;
  }
  return [];
};

const normalizeSectionConfigs = (response: unknown, section: string): BackendConfig[] =>
  getSectionItems(response, section)
    .map((item) => {
      const record = asRecord(item);
      return {
        key: readString(record, 'name', 'Name', 'key', 'Key'),
        value: (record.value ?? record.Value ?? null) as ConfigValue,
        type: readString(record, 'type', 'Type') || 'text',
        description: readString(record, 'description', 'Description'),
        editable: record.editable === undefined && record.Editable === undefined
          ? true
          : readBoolean(record, 'editable', 'Editable'),
      };
    })
    .filter((config) => config.key);

const readString = (record: Record<string, unknown>, ...keys: string[]) => {
  for (const key of keys) {
    const value = record[key];
    if (value !== undefined && value !== null) return String(value);
  }
  return '';
};

const readBoolean = (record: Record<string, unknown>, ...keys: string[]) => {
  for (const key of keys) {
    if (typeof record[key] === 'boolean') return record[key] as boolean;
  }
  return false;
};


const normalizeWorker = (item: unknown): Worker => {
  const record = asRecord(item);
  const name = readString(record, 'name', 'Name');
  const running = readBoolean(record, 'running', 'Running');
  return {
    id: name,
    name,
    section: readString(record, 'section', 'Section'),
    enabled: readBoolean(record, 'enabled', 'Enabled'),
    isActive: running,
    status: running ? 'running' : 'stopped',
  };
};

export const backendConfigService = {
  getSection: async (section: string) =>
    normalizeSectionConfigs(
      await api.get<unknown>(`/v1/config/sections?items=${encodeURIComponent(section)}`),
      section
    ),
  updateSection: (section: string, values: Record<string, ConfigValue>) =>
    api.patch(`/v1/config/sections/${encodeURIComponent(section)}`, { values }),
};

export const workerService = {
  getAll: async () => {
    const response = parseResponse(await api.get<unknown>('/v1/workers'));
    const list = Array.isArray(response)
      ? response
      : (asRecord(response).workers ?? asRecord(response).Workers ?? asRecord(response).data ?? []);
    return (Array.isArray(list) ? list : [])
      .map(normalizeWorker)
      .filter((worker) => worker.name);
  },
  setEnabled: (workerName: string, enabled: boolean) =>
    api.put(`/v1/workers/${encodeURIComponent(workerName)}/enabled`, { enabled }),
  getLogs: async (componentName: string): Promise<string> => {
    const response = parseResponse(
      await api.get<unknown>(`/v1/logs/${encodeURIComponent(componentName)}`)
    );
    if (typeof response === 'string') return response;
    if (Array.isArray(response)) {
      return response
        .map((entry) => {
          const data = asRecord(entry).data;
          if (data === undefined || data === null) return '';
          return typeof data === 'string' ? data : JSON.stringify(data);
        })
        .filter(Boolean)
        .join('\n');
    }
    const data = asRecord(response).data;
    if (data !== undefined && data !== null) {
      return typeof data === 'string' ? data : JSON.stringify(data);
    }
    return '';
  },
  getSettings: async (section: string) => backendConfigService.getSection(section),
  updateSettings: (section: string, settings: Record<string, ConfigValue>) =>
    backendConfigService.updateSection(section, settings),
};