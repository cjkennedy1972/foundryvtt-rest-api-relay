/**
 * Helper for running tests across multiple Foundry versions
 */

import { getGlobalVariable, setGlobalVariable } from './globalVariables';
import { makeRequest } from './apiRequest';
import { ApiRequestConfig } from './apiRequest';

// Check if using existing sessions from env
const useExistingSession = process.env.USE_EXISTING_SESSION === 'true';

/**
 * Get all configured Foundry versions from environment
 */
export function getConfiguredVersions(): string[] {
  const versions = (process.env.TEST_FOUNDRY_VERSIONS || '13')
    .split(',')
    .map(v => v.trim())
    .filter(v => v.length > 0);
  return versions;
}

async function recoverClientFromRelay(version: string): Promise<string> {
  const baseUrl = process.env.TEST_BASE_URL || 'http://localhost:3010';
  const apiKey = process.env.TEST_API_KEY || '';
  if (!apiKey) {
    console.warn(`No TEST_API_KEY set and no cached clientId found for v${version}`);
    return '';
  }

  try {
    const requestConfig: ApiRequestConfig = {
      url: { raw: `${baseUrl}/clients`, host: [baseUrl], path: ['clients'] },
      method: 'GET',
      header: [{ key: 'x-api-key', value: apiKey, type: 'text' }],
    };
    const res = await makeRequest(requestConfig);
    const clients = res.data?.clients || [];
    const match = clients.find((c: any) => `${c.foundryVersion}` === `${version}` && c.clientId);
    const clientId = match?.clientId || clients?.[0]?.clientId || '';
    if (clientId) {
      setGlobalVariable(version, 'clientId', clientId);
      if (match?.systemId) setGlobalVariable(version, 'systemId', match.systemId);
      if (match?.worldId) setGlobalVariable(version, 'worldId', match.worldId);
      return clientId;
    }
  } catch (error) {
    console.warn(`Failed to recover clientId for v${version} from /clients:`, error);
  }

  console.warn(`No clientId found for v${version}`);
  return '';
}

/**
 * Resolve a client ID for a version, recovering it from the relay if needed.
 */
export async function resolveClientId(version?: string): Promise<string> {
  const targetVersion = version || getConfiguredVersions()[0];

  if (useExistingSession) {
    return process.env[`TEST_CLIENT_ID_V${targetVersion}`] || '';
  }

  const cached = getGlobalVariable(targetVersion, 'clientId') || '';
  if (cached) {
    return cached;
  }

  return recoverClientFromRelay(targetVersion);
}

export function hasCachedClientId(version?: string): boolean {
  const targetVersion = version || getConfiguredVersions()[0];
  if (useExistingSession) {
    return !!process.env[`TEST_CLIENT_ID_V${targetVersion}`];
  }
  return !!getGlobalVariable(targetVersion, 'clientId');
}

/**
 * Run a test suite for each configured Foundry version
 */
export function forEachVersion(
  testFn: (version: string, getClientId: () => string) => void
): void {
  const versions = getConfiguredVersions();

  versions.forEach(version => {
    const getClientId = (): string => {
      if (useExistingSession) {
        const envClientId = process.env[`TEST_CLIENT_ID_V${version}`];
        if (envClientId) return envClientId;
        console.warn(`USE_EXISTING_SESSION=true but TEST_CLIENT_ID_V${version} not set`);
        return '';
      }

      const persisted = getGlobalVariable(version, 'clientId');
      if (!persisted) {
        console.warn(`No clientId found for v${version}`);
      }
      return persisted || '';
    };

    testFn(version, getClientId);
  });
}

/**
 * Get client ID for a specific version or default (sync cached lookup only)
 */
export function getClientId(version?: string): string {
  const targetVersion = version || getConfiguredVersions()[0];

  if (useExistingSession) {
    return process.env[`TEST_CLIENT_ID_V${targetVersion}`] || '';
  }

  return getGlobalVariable(targetVersion, 'clientId') || '';
}

// TODO: scrutinize the following methods, and implement system specific tests
/**
 * Get system ID for a specific version
 */
export function getSystemId(version?: string): string {
  const targetVersion = version || getConfiguredVersions()[0];

  if (useExistingSession) {
    const override = process.env[`TEST_SYSTEM_ID_V${targetVersion}`];
    if (override) {
      return override;
    }
  }

  return getGlobalVariable(targetVersion, 'systemId') || '';
}

/**
 * Check if a specific version is running a particular system
 */
export function isSystem(systemId: string, version?: string): boolean {
  return getSystemId(version) === systemId;
}

/**
 * Check if any configured version is running a particular system
 */
export function hasSystemVersion(systemId: string): boolean {
  const versions = getConfiguredVersions();
  return versions.some(v => getSystemId(v) === systemId);
}

/**
 * Get only the versions running a specific system
 */
export function getVersionsWithSystem(systemId: string): string[] {
  const versions = getConfiguredVersions();
  return versions.filter(v => getSystemId(v) === systemId);
}

/**
 * Run a test suite for each configured Foundry version that has a specific system
 */
export function forEachVersionWithSystem(
  systemId: string,
  testFn: (version: string, getClientId: () => string) => void
): void {
  const versions = getVersionsWithSystem(systemId);

  if (versions.length === 0) {
    describe.skip(`(No clients with ${systemId} system)`, () => {
      test('skipped', () => {});
    });
    return;
  }

  versions.forEach(version => {
    const getClientIdFn = (): string => {
      if (useExistingSession) {
        const envClientId = process.env[`TEST_CLIENT_ID_V${version}`];
        if (envClientId) {
          return envClientId;
        }
        console.warn(`USE_EXISTING_SESSION=true but TEST_CLIENT_ID_V${version} not set`);
        return '';
      }

      const clientId = getGlobalVariable(version, 'clientId');
      if (!clientId) {
        console.warn(`No clientId found for v${version}`);
      }
      return clientId || '';
    };

    testFn(version, getClientIdFn);
  });
}
