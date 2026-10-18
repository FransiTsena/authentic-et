import type {
  Payment,
  VerificationStats,
  APIKey,
  Order,
  ProvisioningJob,
  User,
  AuditEvent,
  VerificationResult,
  SystemReady,
  DBStatus,
  DBStatusResponse,
  CleanupReport,
  PlatformSettings,
} from '../types';

declare global {
  interface Window {
    __ENV__?: {
      VITE_API_BASE_URL?: string;
    };
  }
}

const BASE_URL = window.__ENV__?.VITE_API_BASE_URL || '/api/v1';

class APIClient {
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('paygate_token');
  }

  setToken(token: string | null) {
    this.token = token;
    if (token) {
      localStorage.setItem('paygate_token', token);
    } else {
      localStorage.removeItem('paygate_token');
    }
  }

  getToken(): string | null {
    return this.token;
  }

  private async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...(options.headers as Record<string, string>),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${BASE_URL}${path}`, {
      ...options,
      headers,
    });

    if (response.status === 401 && !path.includes('/auth/login')) {
      this.setToken(null);
      window.dispatchEvent(new CustomEvent('paygate_auth_expired'));
    }

    if (!response.ok) {
      let errorMsg = `HTTP Error ${response.status}`;
      try {
        const data = await response.json();
        errorMsg = data.message || data.error || errorMsg;
      } catch {
        // ignore json parse error
      }
      throw new Error(errorMsg);
    }

    return response.json();
  }

  // System & Health & Database
  async getSystemReady(): Promise<SystemReady> {
    const res = await fetch('/ready');
    return res.json();
  }

  async getDatabaseStatus(): Promise<DBStatusResponse> {
    return this.request<DBStatusResponse>('/public/database/status');
  }

  async testDatabaseConnection(driver: string, dsn: string): Promise<{ success: boolean; message?: string; error?: string }> {
    return this.request('/database/test', {
      method: 'POST',
      body: JSON.stringify({ driver, dsn }),
    });
  }

  async configureDatabase(driver: string, dsn: string): Promise<{ success: boolean; message: string; status: DBStatus }> {
    return this.request('/database/configure', {
      method: 'POST',
      body: JSON.stringify({ driver, dsn }),
    });
  }

  async triggerDatabaseCleanup(dryRun = false): Promise<{ success: boolean; message: string; report: CleanupReport }> {
    return this.request('/database/cleanup', {
      method: 'POST',
      body: JSON.stringify({ dryRun }),
    });
  }

  // Auth
  async getSetupStatus(): Promise<{ needsSetup: boolean; allowRegistration: boolean; userCount: number }> {
    return this.request('/auth/setup-status');
  }

  async login(email: string, password: string): Promise<{ token: string; user: User }> {
    const res = await this.request<{ token: string; user: User }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    this.setToken(res.token);
    return res;
  }

  async register(email: string, password: string, name: string): Promise<{ token: string; user: User }> {
    const res = await this.request<{ token: string; user: User }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name }),
    });
    this.setToken(res.token);
    return res;
  }

  async getMe(): Promise<User> {
    return this.request<User>('/auth/me');
  }

  // Payments & Public Verification
  async verifyPayment(payload: {
    provider: string;
    transactionId: string;
    expectedAmount?: number;
    referenceId?: string;
    autoClaim?: boolean;
    claimedBy?: string;
    apiKey?: string;
  }): Promise<VerificationResult> {
    const headers: Record<string, string> = {};
    if (payload.apiKey) {
      headers['X-API-KEY'] = payload.apiKey;
    }

    return this.request<VerificationResult>('/public/payment/verify', {
      method: 'POST',
      headers,
      body: JSON.stringify(payload),
    });
  }

  async claimPayment(payload: {
    provider: string;
    transactionId: string;
    claimedBy?: string;
    apiKey?: string;
  }): Promise<any> {
    const headers: Record<string, string> = {};
    if (payload.apiKey) {
      headers['X-API-KEY'] = payload.apiKey;
    }

    return this.request<any>('/public/payment/claim', {
      method: 'POST',
      headers,
      body: JSON.stringify(payload),
    });
  }

  async getProviders(): Promise<Array<{ key: string; displayName: string }>> {
    return this.request<Array<{ key: string; displayName: string }>>('/public/providers');
  }

  async getStats(): Promise<VerificationStats> {
    return this.request<VerificationStats>('/stats');
  }

  async listTransactions(params: Record<string, any> = {}): Promise<{
    items: Payment[];
    total: number;
    limit: number;
    offset: number;
  }> {
    const q = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== null && v !== '') {
        q.set(k, String(v));
      }
    }
    return this.request(`/transactions?${q.toString()}`);
  }

  async reconcileTransaction(paymentId: string): Promise<Payment> {
    return this.request<Payment>(`/transactions/${paymentId}/reconcile`, {
      method: 'POST',
    });
  }

  // API Keys
  async listAPIKeys(): Promise<APIKey[]> {
    return this.request<APIKey[]>('/api-keys');
  }

  async createAPIKey(payload: {
    name: string;
    tier: string;
    monthlyQuota: number;
    dailyVolumeLimit: number;
    maxRps: number;
  }): Promise<{ key: string; record: APIKey }> {
    return this.request('/api-keys', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async revokeAPIKey(id: string): Promise<{ success: boolean }> {
    return this.request(`/api-keys/${id}`, {
      method: 'DELETE',
    });
  }

  async regenerateAPIKey(id: string): Promise<{ key: string; record: APIKey }> {
    return this.request(`/api-keys/${id}/regenerate`, {
      method: 'POST',
    });
  }

  // Orders
  async listOrders(params: Record<string, any> = {}): Promise<{ items: Order[]; total: number }> {
    const q = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== null && v !== '') q.set(k, String(v));
    }
    return this.request(`/orders?${q.toString()}`);
  }

  async createOrder(payload: {
    serviceType: string;
    tier: string;
    amount: number;
    billingPeriod: number;
  }): Promise<Order> {
    return this.request('/orders', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async verifyOrder(orderId: string, provider: string, transactionId: string): Promise<any> {
    return this.request(`/orders/${orderId}/verify-transaction`, {
      method: 'POST',
      body: JSON.stringify({ provider, transactionId }),
    });
  }

  // Provisioning Jobs
  async listProvisioningJobs(status?: string): Promise<{ items: ProvisioningJob[]; total: number }> {
    const q = status ? `?status=${status}` : '';
    return this.request(`/provisioning/jobs${q}`);
  }

  async retryProvisioningJob(id: string): Promise<{ success: boolean; message: string }> {
    return this.request(`/provisioning/jobs/${id}/retry`, {
      method: 'POST',
    });
  }

  // Audit Logs
  async listAuditLogs(params: Record<string, any> = {}): Promise<{ items: AuditEvent[]; total: number }> {
    const q = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== null && v !== '') q.set(k, String(v));
    }
    return this.request(`/audit/logs?${q.toString()}`);
  }

  // Settings
  async getSettings(): Promise<PlatformSettings> {
    return this.request('/settings');
  }

  async updateSettings(settings: Partial<PlatformSettings>): Promise<{ success: boolean; message: string; settings: PlatformSettings }> {
    return this.request('/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    });
  }

  // Public Health
  async getHealth(): Promise<{ status: string; service: string; uptime: string }> {
    const res = await fetch('/health');
    return res.json();
  }
}

export const api = new APIClient();
