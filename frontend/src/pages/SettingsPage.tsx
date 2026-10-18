import React, { useEffect, useState } from 'react';
import {
  Sliders,
  Save,
  RefreshCw,
  Trash2,
  Sparkles,
  CheckCircle2,
  AlertTriangle,
  Clock,
  ShieldCheck,
  Server,
  Zap,
  Lock,
} from 'lucide-react';
import type { PlatformSettings, CleanupReport } from '../types';
import { api } from '../api/client';

export const SettingsPage: React.FC = () => {
  const [settings, setSettings] = useState<PlatformSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Manual cleanup execution state
  const [cleaning, setCleaning] = useState(false);
  const [cleanupReport, setCleanupReport] = useState<CleanupReport | null>(null);
  const [cleanupError, setCleanupError] = useState<string | null>(null);

  const fetchSettings = async () => {
    setLoading(true);
    try {
      const data = await api.getSettings();
      setSettings(data);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch settings');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSettings();
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!settings) return;
    setSaving(true);
    setError(null);
    setSaveSuccess(false);

    try {
      const res = await api.updateSettings(settings);
      setSettings(res.settings);
      setSaveSuccess(true);
      setTimeout(() => setSaveSuccess(false), 3000);
    } catch (err: any) {
      setError(err.message || 'Failed to update settings');
    } finally {
      setSaving(false);
    }
  };

  const handleManualCleanup = async (dryRun: boolean) => {
    setCleaning(true);
    setCleanupReport(null);
    setCleanupError(null);
    try {
      const res = await api.triggerDatabaseCleanup(dryRun);
      if (res.success) {
        setCleanupReport(res.report);
      } else {
        setCleanupError(res.message || 'Cleanup operation returned false');
      }
    } catch (err: any) {
      setCleanupError(err.message || 'Cleanup execution failed');
    } finally {
      setCleaning(false);
    }
  };

  if (loading) {
    return (
      <div className="py-20 text-center text-xs text-slate-500 font-mono">
        Loading platform settings...
      </div>
    );
  }

  if (!settings) {
    return (
      <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs">
        {error || 'Settings could not be loaded.'}
      </div>
    );
  }

  return (
    <form onSubmit={handleSave} className="space-y-6 max-w-4xl">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold text-white">Settings</h1>
        </div>

        <div className="flex items-center gap-3">
          {saveSuccess && (
            <span className="flex items-center gap-1.5 text-xs text-[#00D26A] font-medium animate-fade-in">
              <CheckCircle2 className="w-3.5 h-3.5" />
              Saved
            </span>
          )}
          <button
            type="submit"
            disabled={saving}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-[#00D26A] hover:bg-[#25F28A] text-black font-semibold text-xs transition disabled:opacity-40"
          >
            {saving ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Save className="w-3.5 h-3.5" />}
            Save Changes
          </button>
        </div>
      </div>

      {error && (
        <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 shrink-0 text-rose-400" />
          <span>{error}</span>
        </div>
      )}

      {/* 1. Automated Data Retention & Pruning */}
      <div className="bg-[#121614] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div className="flex items-center justify-between border-b border-white/[0.06] pb-3">
          <div>
            <h2 className="text-sm font-semibold text-white">Data Retention & Pruning</h2>
            <p className="text-xs text-slate-400 mt-0.5">
              Configure background retention intervals and automated lifecycle rules.
            </p>
          </div>
          <label className="flex items-center gap-2 cursor-pointer select-none">
            <span className="text-xs text-slate-400">Automated Worker</span>
            <input
              type="checkbox"
              checked={settings.retention.cleanupEnabled}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  retention: { ...settings.retention, cleanupEnabled: e.target.checked },
                })
              }
              className="w-4 h-4 rounded bg-black/40 border-white/[0.1] text-[#00D26A] focus:ring-0 focus:ring-offset-0 cursor-pointer"
            />
          </label>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4 pt-1">
          {/* Interval Hours */}
          <div className="space-y-1.5">
            <label className="text-xs text-slate-300 font-medium">Worker Interval (Hours)</label>
            <input
              type="number"
              min="1"
              value={settings.retention.cleanupIntervalHours}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  retention: {
                    ...settings.retention,
                    cleanupIntervalHours: parseInt(e.target.value) || 24,
                  },
                })
              }
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white focus:outline-none focus:border-[#00D26A]"
            />
            <span className="text-[10px] text-slate-500">Run cadence (e.g. 24 for once daily)</span>
          </div>

          {/* Unclaimed Expiry Hours */}
          <div className="space-y-1.5">
            <label className="text-xs text-slate-300 font-medium">Unclaimed Payment Expiry (Hours)</label>
            <input
              type="number"
              min="1"
              value={settings.retention.unclaimedExpiryHours}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  retention: {
                    ...settings.retention,
                    unclaimedExpiryHours: parseInt(e.target.value) || 72,
                  },
                })
              }
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white focus:outline-none focus:border-[#00D26A]"
            />
            <span className="text-[10px] text-slate-500">Unsettled slips marked EXPIRED</span>
          </div>

          {/* Failed Records Days */}
          <div className="space-y-1.5">
            <label className="text-xs text-slate-300 font-medium">Failed Records Retention (Days)</label>
            <input
              type="number"
              min="1"
              value={settings.retention.failedRetentionDays}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  retention: {
                    ...settings.retention,
                    failedRetentionDays: parseInt(e.target.value) || 30,
                  },
                })
              }
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white focus:outline-none focus:border-[#00D26A]"
            />
            <span className="text-[10px] text-slate-500">Purges rejected / invalid slips</span>
          </div>

          {/* Settled Claimed Retention Days */}
          <div className="space-y-1.5">
            <label className="text-xs text-slate-300 font-medium">Settled Payments Retention (Days)</label>
            <input
              type="number"
              min="0"
              value={settings.retention.transactionRetentionDays}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  retention: {
                    ...settings.retention,
                    transactionRetentionDays: parseInt(e.target.value) || 0,
                  },
                })
              }
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white focus:outline-none focus:border-[#00D26A]"
            />
            <span className="text-[10px] text-slate-500">Set 0 to retain financial ledger forever</span>
          </div>

          {/* Completed Jobs Days */}
          <div className="space-y-1.5">
            <label className="text-xs text-slate-300 font-medium">Completed Outbox Jobs (Days)</label>
            <input
              type="number"
              min="1"
              value={settings.retention.completedJobsRetentionDays}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  retention: {
                    ...settings.retention,
                    completedJobsRetentionDays: parseInt(e.target.value) || 14,
                  },
                })
              }
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white focus:outline-none focus:border-[#00D26A]"
            />
            <span className="text-[10px] text-slate-500">Dispatched provisioning task history</span>
          </div>

          {/* Audit Logs Days */}
          <div className="space-y-1.5">
            <label className="text-xs text-slate-300 font-medium">Audit Trail Retention (Days)</label>
            <input
              type="number"
              min="1"
              value={settings.retention.auditRetentionDays}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  retention: {
                    ...settings.retention,
                    auditRetentionDays: parseInt(e.target.value) || 180,
                  },
                })
              }
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white focus:outline-none focus:border-[#00D26A]"
            />
            <span className="text-[10px] text-slate-500">Immutable admin audit event log</span>
          </div>
        </div>

        {/* Dry run checkbox & Manual actions */}
        <div className="pt-3 border-t border-white/[0.06] flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
          <label className="flex items-center gap-2 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={settings.retention.dryRunCleanup}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  retention: { ...settings.retention, dryRunCleanup: e.target.checked },
                })
              }
              className="w-4 h-4 rounded bg-black/40 border-white/[0.1] text-[#00D26A] focus:ring-0 focus:ring-offset-0 cursor-pointer"
            />
            <span className="text-xs text-slate-300">Dry Run Mode (simulate queries without deletion)</span>
          </label>

          <div className="flex items-center gap-2 self-end sm:self-auto">
            <button
              type="button"
              onClick={() => handleManualCleanup(true)}
              disabled={cleaning}
              className="px-3 py-1.5 bg-white/[0.04] hover:bg-white/[0.08] disabled:opacity-40 text-slate-200 text-xs font-medium rounded-lg flex items-center gap-1.5 border border-white/[0.06] transition"
            >
              {cleaning ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Sparkles className="w-3 h-3 text-amber-400" />}
              Dry Run Test
            </button>
            <button
              type="button"
              onClick={() => handleManualCleanup(false)}
              disabled={cleaning}
              className="px-3 py-1.5 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/20 disabled:opacity-40 text-rose-300 text-xs font-medium rounded-lg flex items-center gap-1.5 transition"
            >
              {cleaning ? <RefreshCw className="w-3 h-3 animate-spin" /> : <Trash2 className="w-3 h-3" />}
              Run Cleanup Now
            </button>
          </div>
        </div>

        {/* Cleanup Result Feedback */}
        {cleanupError && (
          <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 text-rose-400 shrink-0" />
            <span>Cleanup execution failed: {cleanupError}</span>
          </div>
        )}

        {cleanupReport && (
          <div className="p-3.5 rounded-lg bg-black/40 border border-white/[0.04] space-y-2 text-xs">
            <div className="flex items-center justify-between font-semibold text-white">
              <span className="flex items-center gap-1.5 text-[#00D26A]">
                <CheckCircle2 className="w-3.5 h-3.5" />
                Cleanup Finished {cleanupReport.dryRun ? '(Dry Run Simulation)' : '(Pruned Records)'}
              </span>
              <span className="font-mono text-slate-500 text-[11px]">
                {new Date(cleanupReport.executedAt).toLocaleTimeString()}
              </span>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-5 gap-2 pt-1 font-mono text-xs">
              <div className="bg-[#121614] p-2 rounded border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Expired Unclaimed</div>
                <div className="font-bold text-amber-400">{cleanupReport.expiredUnclaimedPayments}</div>
              </div>
              <div className="bg-[#121614] p-2 rounded border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Purged Failed</div>
                <div className="font-bold text-rose-400">{cleanupReport.deletedFailedPayments}</div>
              </div>
              <div className="bg-[#121614] p-2 rounded border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Purged Claimed</div>
                <div className="font-bold text-slate-300">{cleanupReport.deletedClaimedPayments}</div>
              </div>
              <div className="bg-[#121614] p-2 rounded border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Purged Jobs</div>
                <div className="font-bold text-slate-300">{cleanupReport.deletedCompletedJobs}</div>
              </div>
              <div className="bg-[#121614] p-2 rounded border border-white/[0.06]">
                <div className="text-slate-400 text-[10px]">Purged Audit</div>
                <div className="font-bold text-slate-300">{cleanupReport.deletedAuditLogs}</div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* 2. Verification Rules & Tolerances */}
      <div className="bg-[#121614] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-white">Payment Verification Rules</h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Define amount match tolerances and merchant receiver verification parameters.
          </p>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="space-y-1.5">
            <label className="text-xs text-slate-300 font-medium">Underpay Tolerance (ETB)</label>
            <input
              type="number"
              step="any"
              min="0"
              value={settings.verification.underpayTolerance}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  verification: {
                    ...settings.verification,
                    underpayTolerance: parseFloat(e.target.value) || 0,
                  },
                })
              }
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white focus:outline-none focus:border-[#00D26A]"
            />
            <span className="text-[10px] text-slate-500">Allowed deficit in ETB (e.g. 1.0)</span>
          </div>

          <div className="space-y-1.5">
            <label className="text-xs text-slate-300 font-medium">Overpay Tolerance (ETB)</label>
            <input
              type="number"
              step="any"
              min="0"
              value={settings.verification.overpayTolerance}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  verification: {
                    ...settings.verification,
                    overpayTolerance: parseFloat(e.target.value) || 0,
                  },
                })
              }
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white focus:outline-none focus:border-[#00D26A]"
            />
            <span className="text-[10px] text-slate-500">Allowed excess in ETB (e.g. 5.0)</span>
          </div>

          <div className="space-y-1.5 flex flex-col justify-center pt-2">
            <label className="flex items-center gap-2 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={settings.verification.mockMode}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    verification: {
                      ...settings.verification,
                      mockMode: e.target.checked,
                    },
                  })
                }
                className="w-4 h-4 rounded bg-black/40 border-white/[0.1] text-[#00D26A] focus:ring-0 focus:ring-offset-0 cursor-pointer"
              />
              <span className="text-xs text-slate-300 font-medium">Sandbox Mock Mode</span>
            </label>
            <span className="text-[10px] text-slate-500 mt-1">Allows TEST_SUCCESS slips for testing</span>
          </div>
        </div>
      </div>

      {/* 3. Security & Access */}
      <div className="bg-[#121614] border border-white/[0.06] rounded-xl p-5 space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-white">Security & Access</h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Platform authentication parameters and self-registration controls.
          </p>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="flex items-center justify-between p-3 rounded-lg bg-black/40 border border-white/[0.04]">
            <div>
              <div className="text-xs font-medium text-white">Open Registration</div>
              <div className="text-[10px] text-slate-500">Allow new operators to register accounts</div>
            </div>
            <input
              type="checkbox"
              checked={settings.security.allowRegistration}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  security: {
                    ...settings.security,
                    allowRegistration: e.target.checked,
                  },
                })
              }
              className="w-4 h-4 rounded bg-black/40 border-white/[0.1] text-[#00D26A] focus:ring-0 focus:ring-offset-0 cursor-pointer"
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-xs text-slate-300 font-medium">Session Expiry (Hours)</label>
            <input
              type="number"
              min="1"
              value={settings.security.jwtExpiryHours}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  security: {
                    ...settings.security,
                    jwtExpiryHours: parseInt(e.target.value) || 72,
                  },
                })
              }
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white focus:outline-none focus:border-[#00D26A]"
            />
            <span className="text-[10px] text-slate-500">JWT login token lifetime</span>
          </div>
        </div>
      </div>
    </form>
  );
};
