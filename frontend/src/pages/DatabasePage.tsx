import React, { useState, useEffect } from 'react';
import { api } from '../api/client';
import type { DBStatus, RetentionInfo, CleanupReport } from '../types';
import {
  Database,
  AlertTriangle,
  CheckCircle2,
  Server,
  HardDrive,
  RefreshCw,
  Cpu,
  ShieldCheck,
  ArrowRight,
  Terminal,
  Activity,
  Trash2,
  Clock,
  Sparkles,
  ExternalLink,
  Layers,
} from 'lucide-react';

export const DatabasePage: React.FC = () => {
  const [status, setStatus] = useState<DBStatus | null>(null);
  const [retention, setRetention] = useState<RetentionInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [driver, setDriver] = useState<'postgres' | 'sqlite'>('sqlite');

  // Form states
  const [postgresUrl, setPostgresUrl] = useState('');
  const [sqlitePath, setSqlitePath] = useState('./data/paygate.db');

  // Test & configure states
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message?: string; error?: string } | null>(null);
  const [applying, setApplying] = useState(false);
  const [applyResult, setApplyResult] = useState<{ success: boolean; message?: string; error?: string } | null>(null);

  // Retention cleanup states
  const [cleaning, setCleaning] = useState(false);
  const [cleanupReport, setCleanupReport] = useState<CleanupReport | null>(null);
  const [cleanupError, setCleanupError] = useState<string | null>(null);

  const fetchStatus = async () => {
    try {
      setLoading(true);
      const res = await api.getDatabaseStatus();
      const db = res.database || (res as any);
      setStatus(db);
      if (res.retention) {
        setRetention(res.retention);
      }
      if (db.driver === 'postgres' || db.driver === 'sqlite') {
        setDriver(db.driver);
      }
    } catch (err: any) {
      console.error('Failed to fetch DB status', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
  }, []);

  const handleTest = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const dsn = driver === 'postgres' ? postgresUrl : sqlitePath;
      if (!dsn.trim()) {
        setTestResult({ success: false, error: 'Connection string or path is required' });
        return;
      }
      const res = await api.testDatabaseConnection(driver, dsn);
      setTestResult(res);
    } catch (err: any) {
      setTestResult({ success: false, error: err.message || 'Connection test failed' });
    } finally {
      setTesting(false);
    }
  };

  const handleApply = async () => {
    setApplying(true);
    setApplyResult(null);
    try {
      const dsn = driver === 'postgres' ? postgresUrl : sqlitePath;
      if (!dsn.trim()) {
        setApplyResult({ success: false, error: 'Connection string or path is required' });
        return;
      }
      const res = await api.configureDatabase(driver, dsn);
      setApplyResult({ success: true, message: res.message });
      setStatus(res.status);
    } catch (err: any) {
      setApplyResult({ success: false, error: err.message || 'Failed to switch database' });
    } finally {
      setApplying(false);
    }
  };

  const handleCleanup = async (dryRun = false) => {
    setCleaning(true);
    setCleanupReport(null);
    setCleanupError(null);
    try {
      const res = await api.triggerDatabaseCleanup(dryRun);
      if (res.success) {
        setCleanupReport(res.report);
      } else {
        setCleanupError(res.message || 'Cleanup returned failure');
      }
    } catch (err: any) {
      setCleanupError(err.message || 'Failed to trigger database cleanup');
    } finally {
      setCleaning(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold text-white">Database</h1>
        </div>

        <button
          onClick={fetchStatus}
          disabled={loading}
          className="p-2 rounded-lg bg-white/[0.04] text-slate-400 hover:text-white border border-white/[0.06] hover:border-white/[0.12] transition self-start"
          title="Refresh Engine Status"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Warning if Ephemeral Mode */}
      {status?.isEphemeral && (
        <div className="bg-amber-500/10 border border-amber-500/20 rounded-xl p-4 text-xs space-y-2">
          <div className="flex items-center gap-2.5 font-bold text-amber-300">
            <AlertTriangle className="w-4 h-4 text-amber-400 animate-pulse" />
            Operating in Ephemeral In-Memory Storage Mode
          </div>
          <p className="text-amber-200/90 leading-relaxed">
            {status.warningMessage ||
              'No persistent database is currently connected. Transactions, API keys, and audit logs are retained in memory and will NOT persist across restarts!'}
          </p>
          <div className="text-amber-400 font-semibold">
            Recommended Action: Switch to the default SQLite path (./data/paygate.db) or connect to your external PostgreSQL server below.
          </div>
        </div>
      )}

      {/* Engine Status Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Driver */}
        <div className="bg-[#121614] border border-white/[0.06] rounded-xl p-4 space-y-2">
          <div className="flex items-center justify-between text-slate-400 text-xs font-semibold uppercase tracking-wider">
            <span>Active Storage Driver</span>
            <Cpu className="w-4 h-4 text-[#00D26A]" />
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xl font-bold uppercase text-white font-mono">
              {status?.driver || 'Loading...'}
            </span>
            {status?.isEphemeral ? (
              <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">
                Memory Only
              </span>
            ) : (
              <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-[#00D26A]/10 text-[#00D26A] border border-[#00D26A]/20">
                ACID Persistent
              </span>
            )}
          </div>
          <p className="text-xs text-slate-400">
            {status?.driver === 'sqlite'
              ? 'Zero-dependency embedded database in WAL mode.'
              : status?.driver === 'postgres'
              ? 'Multi-node high throughput PostgreSQL instance.'
              : 'Volatile in-memory mock engine.'}
          </p>
        </div>

        {/* Health */}
        <div className="bg-[#121614] border border-white/[0.06] rounded-xl p-4 space-y-2">
          <div className="flex items-center justify-between text-slate-400 text-xs font-semibold uppercase tracking-wider">
            <span>Connection Health</span>
            <Activity className="w-4 h-4 text-[#00D26A]" />
          </div>
          <div className="flex items-center gap-2">
            <div
              className={`w-2.5 h-2.5 rounded-full ${
                status?.connected ? 'bg-[#00D26A] animate-pulse' : 'bg-rose-500'
              }`}
            />
            <span className="text-xl font-bold text-white">
              {status?.connected ? 'Healthy & Online' : 'Disconnected'}
            </span>
          </div>
          <p className="text-xs text-slate-400 font-mono truncate" title={status?.maskedDsn}>
            {status?.maskedDsn ? `DSN: ${status.maskedDsn}` : 'Engine ready'}
          </p>
        </div>

        {/* Concurrency Safe */}
        <div className="bg-[#121614] border border-white/[0.06] rounded-xl p-4 space-y-2">
          <div className="flex items-center justify-between text-slate-400 text-xs font-semibold uppercase tracking-wider">
            <span>Concurrency Architecture</span>
            <ShieldCheck className="w-4 h-4 text-[#00D26A]" />
          </div>
          <div className="text-xl font-bold text-white">
            {status?.driver === 'postgres'
              ? 'Multi-Replica Safe'
              : status?.driver === 'sqlite'
              ? 'Single-Node WAL'
              : 'Ephemeral Mode'}
          </div>
          <p className="text-xs text-slate-400">
            {status?.driver === 'postgres'
              ? 'Row-level locking with SKIP LOCKED for worker pools.'
              : status?.driver === 'sqlite'
              ? 'Concurrent read transactions with single atomic writer.'
              : 'Standalone test & preview mode.'}
          </p>
        </div>
      </div>

      {/* Database Switcher & Configuration */}
      <div className="bg-[#121614] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-white">Configuration</h2>
        </div>

        {/* Driver Tabs */}
        <div className="flex gap-2 border-b border-white/[0.06] pb-3">
          <button
            type="button"
            onClick={() => {
              setDriver('sqlite');
              setTestResult(null);
              setApplyResult(null);
            }}
            className={`flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition ${
              driver === 'sqlite'
                ? 'bg-[#00D26A] text-black font-bold'
                : 'bg-white/[0.04] text-slate-400 hover:text-white border border-white/[0.06]'
            }`}
          >
            <HardDrive className="w-3.5 h-3.5" />
            SQLite
          </button>
          <button
            type="button"
            onClick={() => {
              setDriver('postgres');
              setTestResult(null);
              setApplyResult(null);
            }}
            className={`flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition ${
              driver === 'postgres'
                ? 'bg-[#00D26A] text-black font-bold'
                : 'bg-white/[0.04] text-slate-400 hover:text-white border border-white/[0.06]'
            }`}
          >
            <Server className="w-3.5 h-3.5" />
            PostgreSQL
          </button>
        </div>

        {/* Input Field */}
        <div className="space-y-3">
          {driver === 'sqlite' ? (
            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">SQLite Database File Path</label>
              <input
                type="text"
                value={sqlitePath}
                onChange={(e) => setSqlitePath(e.target.value)}
                placeholder="./data/paygate.db"
                className="w-full px-3 py-2 bg-black/40 border border-white/[0.08] rounded-lg text-xs text-white font-mono focus:outline-none focus:border-[#00D26A]"
              />
              <p className="text-[11px] text-slate-500">
                Default: <code>./data/paygate.db</code>. The directory and file are created automatically if missing.
              </p>
            </div>
          ) : (
            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">PostgreSQL Connection URL (DSN)</label>
              <input
                type="text"
                value={postgresUrl}
                onChange={(e) => setPostgresUrl(e.target.value)}
                placeholder="postgres://user:password@localhost:5432/paygate?sslmode=disable"
                className="w-full px-3 py-2 bg-black/40 border border-white/[0.08] rounded-lg text-xs text-white font-mono focus:outline-none focus:border-[#00D26A]"
              />
              <p className="text-[11px] text-slate-500">
                Format: <code>postgres://user:password@host:5432/dbname?sslmode=disable</code>
              </p>
            </div>
          )}

          {/* Test & Apply Results */}
          {testResult && (
            <div
              className={`p-3 rounded-lg text-xs flex items-center gap-2 border ${
                testResult.success
                  ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-300'
                  : 'bg-rose-500/10 border-rose-500/20 text-rose-300'
              }`}
            >
              {testResult.success ? (
                <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
              ) : (
                <AlertTriangle className="w-4 h-4 text-rose-400 shrink-0" />
              )}
              <span>{testResult.success ? testResult.message : testResult.error}</span>
            </div>
          )}

          {applyResult && (
            <div
              className={`p-3 rounded-lg text-xs flex items-center gap-2 border ${
                applyResult.success
                  ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-300'
                  : 'bg-rose-500/10 border-rose-500/20 text-rose-300'
              }`}
            >
              {applyResult.success ? (
                <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
              ) : (
                <AlertTriangle className="w-4 h-4 text-rose-400 shrink-0" />
              )}
              <span>{applyResult.success ? applyResult.message : applyResult.error}</span>
            </div>
          )}

          {/* Buttons */}
          <div className="flex items-center gap-2.5 pt-2">
            <button
              type="button"
              onClick={handleTest}
              disabled={testing || applying}
              className="px-3.5 py-2 bg-white/[0.04] hover:bg-white/[0.08] disabled:opacity-40 text-slate-200 text-xs font-semibold rounded-lg flex items-center gap-1.5 border border-white/[0.06] transition"
            >
              {testing ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Terminal className="w-3.5 h-3.5" />}
              Test Connection
            </button>
            <button
              type="button"
              onClick={handleApply}
              disabled={testing || applying}
              className="px-4 py-2 bg-[#00D26A] hover:bg-[#00ba5e] disabled:opacity-40 text-black text-xs font-semibold rounded-lg flex items-center gap-1.5 transition"
            >
              {applying ? <RefreshCw className="w-3.5 h-3.5 animate-spin text-black" /> : <ArrowRight className="w-3.5 h-3.5" />}
              Apply & Switch Database
            </button>
          </div>
        </div>
      </div>

      {/* Automated Data Retention & Maintenance */}
      <div className="bg-[#121614] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-white/[0.06] pb-3">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold text-white">Data Retention</h2>
            {retention?.enabled && (
              <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-[#00D26A]/10 text-[#00D26A]">
                Every {retention.intervalHours}h
              </span>
            )}
            {retention?.dryRun && (
              <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-amber-500/10 text-amber-400">
                Dry Run
              </span>
            )}
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => handleCleanup(true)}
              disabled={cleaning}
              className="px-3 py-1.5 bg-white/[0.04] hover:bg-white/[0.08] disabled:opacity-40 text-slate-200 text-xs font-medium rounded-lg flex items-center gap-1.5 border border-white/[0.06] transition"
            >
              {cleaning ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Sparkles className="w-3 h-3 text-amber-400" />}
              Dry Run
            </button>
            <button
              type="button"
              onClick={() => handleCleanup(false)}
              disabled={cleaning}
              className="px-3.5 py-1.5 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/20 disabled:opacity-40 text-rose-300 text-xs font-medium rounded-lg flex items-center gap-1.5 transition"
            >
              {cleaning ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Trash2 className="w-3 h-3" />}
              Run Cleanup
            </button>
          </div>
        </div>

        {/* Retention Policy Cards */}
        <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
          <div className="p-3 rounded-lg bg-black/40 border border-white/[0.04]">
            <div className="text-xs text-slate-400">Unclaimed Expiry</div>
            <div className="mt-1 text-sm font-bold text-white font-mono">
              {retention?.unclaimedExpiryHours || 72}h
            </div>
          </div>

          <div className="p-3 rounded-lg bg-black/40 border border-white/[0.04]">
            <div className="text-xs text-slate-400">Failed Records</div>
            <div className="mt-1 text-sm font-bold text-white font-mono">
              {retention?.failedRetentionDays || 30}d
            </div>
          </div>

          <div className="p-3 rounded-lg bg-black/40 border border-white/[0.04]">
            <div className="text-xs text-slate-400">Claimed Payments</div>
            <div className="mt-1 text-sm font-bold text-white font-mono">
              {retention && retention.transactionRetentionDays > 0 ? `${retention.transactionRetentionDays}d` : 'Forever'}
            </div>
          </div>

          <div className="p-3 rounded-lg bg-black/40 border border-white/[0.04]">
            <div className="text-xs text-slate-400">Completed Jobs</div>
            <div className="mt-1 text-sm font-bold text-white font-mono">
              {retention?.completedJobsRetentionDays || 14}d
            </div>
          </div>

          <div className="p-3 rounded-lg bg-black/40 border border-white/[0.04]">
            <div className="text-xs text-slate-400">Audit Logs</div>
            <div className="mt-1 text-sm font-bold text-white font-mono">
              {retention?.auditRetentionDays || 180}d
            </div>
          </div>
        </div>

        {/* Cleanup Execution Feedback */}
        {cleanupError && (
          <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 text-rose-400 shrink-0" />
            <span>Cleanup execution failed: {cleanupError}</span>
          </div>
        )}

        {cleanupReport && (
          <div className="p-4 rounded-xl bg-black/40 border border-white/[0.04] space-y-2">
            <div className="flex items-center justify-between text-xs font-semibold text-white">
              <span className="flex items-center gap-1.5 text-[#00D26A]">
                <CheckCircle2 className="w-4 h-4" />
                Cleanup Finished {cleanupReport.dryRun ? '(Dry Run Simulation)' : '(Pruning Applied)'}
              </span>
              <span className="font-mono text-slate-500 text-[11px]">
                {new Date(cleanupReport.executedAt).toLocaleTimeString()}
              </span>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-5 gap-2 pt-1 text-xs">
              <div className="bg-[#121614] p-2.5 rounded-lg border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Expired Unclaimed</div>
                <div className="font-bold text-amber-400 font-mono text-sm">{cleanupReport.expiredUnclaimedPayments}</div>
              </div>
              <div className="bg-[#121614] p-2.5 rounded-lg border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Purged Failed</div>
                <div className="font-bold text-rose-400 font-mono text-sm">{cleanupReport.deletedFailedPayments}</div>
              </div>
              <div className="bg-[#121614] p-2.5 rounded-lg border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Purged Claimed</div>
                <div className="font-bold text-slate-300 font-mono text-sm">{cleanupReport.deletedClaimedPayments}</div>
              </div>
              <div className="bg-[#121614] p-2.5 rounded-lg border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Purged Jobs</div>
                <div className="font-bold text-slate-300 font-mono text-sm">{cleanupReport.deletedCompletedJobs}</div>
              </div>
              <div className="bg-[#121614] p-2.5 rounded-lg border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Purged Audit</div>
                <div className="font-bold text-slate-300 font-mono text-sm">{cleanupReport.deletedAuditLogs}</div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Production Stack Info */}
      <div className="bg-[#121614] border border-white/[0.06] rounded-xl p-5 space-y-2">
        <h3 className="text-sm font-bold text-white flex items-center gap-2">
          <Terminal className="w-4 h-4 text-[#00D26A]" />
          Full Clustered Production Stack: docker-compose-full.yml
        </h3>
        <p className="text-xs text-slate-400">
          To launch PayGate with a dedicated PostgreSQL 16 server, automated persistent volumes, reports storage, and background worker queues, run:
        </p>
        <pre className="bg-black/40 p-3 rounded-lg font-mono text-xs text-[#00D26A] overflow-x-auto border border-white/[0.04]">
          docker compose -f docker-compose-full.yml up --build -d
        </pre>
      </div>
    </div>
  );
};
