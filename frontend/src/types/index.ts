export type PaymentStatus =
  | 'PENDING'
  | 'VERIFYING'
  | 'VERIFIED'
  | 'CLAIMED'
  | 'FAILED'
  | 'EXPIRED'
  | 'REFUNDED';

export interface Payment {
  id: string;
  providerKey: string;
  transactionId: string;
  referenceId: string;
  referenceType: string;
  amount: number;
  currency: string;
  status: PaymentStatus;
  claimed: boolean;
  claimedAt?: string;
  claimedBy?: string;
  apiKeyId?: string;
  apiKeyPrefix?: string;
  apiKeyName?: string;
  ownerUserId?: string;
  payerIdentifier?: string;
  receiverIdentifier?: string;
  rawPayload?: Record<string, any>;
  errorMessage?: string;
  errorCode?: string;
  verifiedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface VerificationStats {
  totalVerifications: number;
  successfulVerifications: number;
  failedVerifications: number;
  successRatePercent: number;
  totalVerifiedAmount: number;
  totalClaimedAmount: number;
  providers: Record<string, number>;
}

export interface APIKey {
  id: string;
  prefix: string;
  name: string;
  ownerEmail: string;
  ownerUserId: string;
  active: boolean;
  tier: 'free' | 'pro' | 'enterprise';
  maxRps: number;
  monthlyQuota: number;
  usedThisMonth: number;
  dailyVolumeLimit: number;
  verifiedVolumeToday: number;
  lastVolumeResetDate: string;
  permissions: string[];
  lastUsedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Order {
  id: string;
  userId: string;
  serviceType: string;
  tier: string;
  amount: number;
  currency: string;
  billingPeriod: number;
  status: string;
  paymentProvider?: string;
  paymentReference?: string;
  paidAt?: string;
  metadata?: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

export interface ProvisioningJob {
  id: string;
  key: string;
  orderId?: string;
  userId?: string;
  serviceType: string;
  action: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'manual_review';
  attempts: number;
  maxAttempts: number;
  lockedAt?: string;
  nextAttemptAt?: string;
  completedAt?: string;
  lastError?: string;
  payload?: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'developer' | 'viewer';
  active: boolean;
  createdAt: string;
}

export interface AuditEvent {
  id: string;
  action: string;
  actorType: string;
  actorId: string;
  resourceId: string;
  resourceType: string;
  details?: Record<string, any>;
  ipAddress?: string;
  correlationId?: string;
  createdAt: string;
}

export interface VerificationResult {
  verified: boolean;
  providerKey: string;
  transactionId: string;
  amount: number;
  currency: string;
  status: string;
  alreadyProcessed: boolean;
  claimed: boolean;
  claimedAt?: string;
  claimedBy?: string;
  payerIdentifier?: string;
  receiverIdentifier?: string;
  timestamp?: string;
  warningMessage?: string;
  errorMessage?: string;
  errorCode?: string;
}

export interface SystemReady {
  status: string;
  database: string;
  driver: 'postgres' | 'sqlite' | 'memory';
  uptime: string;
}

export interface DBStatus {
  driver: 'memory' | 'sqlite' | 'postgres' | string;
  isEphemeral: boolean;
  warningMessage: string;
  connected: boolean;
  connectionError?: string;
  maskedDsn?: string;
  connectedAt: string;
}

export interface RetentionInfo {
  enabled: boolean;
  intervalHours: number;
  transactionRetentionDays: number;
  failedRetentionDays: number;
  unclaimedExpiryHours: number;
  auditRetentionDays: number;
  completedJobsRetentionDays: number;
  dryRun: boolean;
}

export interface CleanupReport {
  expiredUnclaimedPayments: number;
  deletedFailedPayments: number;
  deletedClaimedPayments: number;
  deletedCompletedJobs: number;
  deletedAuditLogs: number;
  dryRun: boolean;
  duration: number;
  executedAt: string;
}

export interface DBStatusResponse {
  database: DBStatus;
  retention?: RetentionInfo;
}

export interface PlatformSettings {
  retention: {
    cleanupEnabled: boolean;
    cleanupIntervalHours: number;
    unclaimedExpiryHours: number;
    failedRetentionDays: number;
    transactionRetentionDays: number;
    completedJobsRetentionDays: number;
    auditRetentionDays: number;
    dryRunCleanup: boolean;
  };
  verification: {
    underpayTolerance: number;
    overpayTolerance: number;
    mockMode: boolean;
    merchantNames: string[];
    merchantAccounts: string[];
  };
  security: {
    allowRegistration: boolean;
    jwtExpiryHours: number;
  };
}

