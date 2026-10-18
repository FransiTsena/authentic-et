import React, { useEffect, useState } from 'react';
import {
  Key,
  Plus,
  Copy,
  Check,
  Trash2,
  RefreshCw,
  AlertTriangle,
  ShieldAlert,
  Zap,
} from 'lucide-react';
import type { APIKey } from '../types';
import { api } from '../api/client';

export const ApiKeysPage: React.FC = () => {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [newRawKey, setNewRawKey] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [tier, setTier] = useState('free');
  const [creating, setCreating] = useState(false);
  const [copiedKey, setCopiedKey] = useState(false);

  const fetchKeys = async () => {
    setLoading(true);
    try {
      const data = await api.listAPIKeys();
      setKeys(data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchKeys();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    try {
      let quota = 1000;
      let dailyLimit = 50000;
      let maxRps = 10;
      if (tier === 'pro') {
        quota = 50000;
        dailyLimit = 150000;
        maxRps = 100;
      } else if (tier === 'enterprise') {
        quota = 1000000;
        dailyLimit = -1;
        maxRps = 500;
      }

      const res = await api.createAPIKey({
        name: name.trim() || 'Default Key',
        tier,
        monthlyQuota: quota,
        dailyVolumeLimit: dailyLimit,
        maxRps,
      });

      setNewRawKey(res.key);
      setIsCreateOpen(false);
      setName('');
      fetchKeys();
    } catch (err: any) {
      alert(`Failed to create API key: ${err.message}`);
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (id: string) => {
    if (!window.confirm('Are you sure you want to revoke this API key? Applications using it will immediately be rejected.')) {
      return;
    }
    try {
      await api.revokeAPIKey(id);
      fetchKeys();
    } catch (err: any) {
      alert(`Failed to revoke key: ${err.message}`);
    }
  };

  const handleRegenerate = async (id: string) => {
    if (!window.confirm('Regenerating this key will immediately revoke the current token. Continue?')) {
      return;
    }
    try {
      const res = await api.regenerateAPIKey(id);
      setNewRawKey(res.key);
      fetchKeys();
    } catch (err: any) {
      alert(`Failed to regenerate key: ${err.message}`);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold text-white">API Keys</h1>
        </div>
        <button
          onClick={() => setIsCreateOpen(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg bg-[#00D26A] text-black hover:bg-[#00ba5e] transition"
        >
          <Plus className="w-3.5 h-3.5" />
          Create Key
        </button>
      </div>

      {/* New Key Created Banner */}
      {newRawKey && (
        <div className="p-5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-200 space-y-3">
          <div className="flex items-center gap-2 font-semibold text-sm text-amber-400">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            Save your API Key now
          </div>
          <p className="text-xs text-amber-200/80">
            This secret key will <strong>never be shown again</strong>. Store it securely in your environment variables.
          </p>
          <div className="flex items-center gap-2 bg-black/40 p-2.5 rounded-lg border border-amber-500/20 font-mono text-xs text-white">
            <span className="flex-1 select-all break-all">{newRawKey}</span>
            <button
              onClick={() => copyToClipboard(newRawKey)}
              className="p-1.5 rounded bg-amber-500/20 text-amber-300 hover:bg-amber-500/30 transition shrink-0"
              title="Copy Key"
            >
              {copiedKey ? <Check className="w-4 h-4 text-[#00D26A]" /> : <Copy className="w-4 h-4" />}
            </button>
          </div>
          <button
            onClick={() => setNewRawKey(null)}
            className="text-[11px] font-semibold text-amber-400 hover:underline"
          >
            I have saved this key safely
          </button>
        </div>
      )}

      {/* Keys List */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {loading ? (
          <div className="col-span-full py-12 text-center text-xs text-slate-500">Loading API keys...</div>
        ) : keys.length === 0 ? (
          <div className="col-span-full py-12 text-center text-xs text-slate-500">
            No API keys created yet. Generate one above to begin integrating!
          </div>
        ) : (
          keys.map((k) => {
            const volumePercent = k.dailyVolumeLimit <= 0 ? 0 : Math.min(100, Math.round((k.verifiedVolumeToday / k.dailyVolumeLimit) * 100));
            const quotaPercent = k.monthlyQuota <= 0 ? 0 : Math.min(100, Math.round((k.usedThisMonth / k.monthlyQuota) * 100));

            return (
              <div
                key={k.id}
                className={`p-5 rounded-xl bg-[#121614] border transition space-y-4 flex flex-col justify-between ${
                  k.active ? 'border-white/[0.06]' : 'border-white/[0.03] opacity-50'
                }`}
              >
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-bold text-white tracking-tight">{k.name}</span>
                    <span
                      className={`text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full border ${
                        k.active
                          ? 'bg-[#00D26A]/10 text-[#00D26A] border-[#00D26A]/20'
                          : 'bg-white/[0.04] text-slate-500 border-white/[0.06]'
                      }`}
                    >
                      {k.active ? k.tier : 'Revoked'}
                    </span>
                  </div>

                  <div className="text-xs font-mono text-slate-400 bg-black/40 p-2 rounded-lg border border-white/[0.04] flex items-center justify-between">
                    <span>{k.prefix}••••••••••••</span>
                    <span className="text-[10px] font-sans text-slate-500">{k.maxRps} RPS max</span>
                  </div>

                  {/* Usage Bars */}
                  <div className="space-y-2 pt-1 text-[11px]">
                    <div>
                      <div className="flex justify-between text-slate-400 mb-1">
                        <span>Daily Volume Limit</span>
                        <span className="font-semibold text-slate-200">
                          {k.verifiedVolumeToday.toLocaleString()} / {k.dailyVolumeLimit === -1 ? 'Unlimited' : `${k.dailyVolumeLimit.toLocaleString()} ETB`}
                        </span>
                      </div>
                      {k.dailyVolumeLimit > 0 && (
                        <div className="h-1.5 w-full bg-black/40 rounded-full overflow-hidden">
                          <div
                            className={`h-full transition-all ${volumePercent > 85 ? 'bg-rose-500' : 'bg-[#00D26A]'}`}
                            style={{ width: `${volumePercent}%` }}
                          ></div>
                        </div>
                      )}
                    </div>

                    <div>
                      <div className="flex justify-between text-slate-400 mb-1">
                        <span>Monthly Verifications</span>
                        <span className="font-semibold text-slate-200">
                          {k.usedThisMonth.toLocaleString()} / {k.monthlyQuota.toLocaleString()}
                        </span>
                      </div>
                      <div className="h-1.5 w-full bg-black/40 rounded-full overflow-hidden">
                        <div
                          className={`h-full transition-all ${quotaPercent > 85 ? 'bg-amber-500' : 'bg-[#00D26A]'}`}
                          style={{ width: `${quotaPercent}%` }}
                        ></div>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Actions */}
                {k.active && (
                  <div className="pt-3 border-t border-white/[0.06] flex items-center justify-end gap-2">
                    <button
                      onClick={() => handleRegenerate(k.id)}
                      className="px-2.5 py-1 text-xs font-medium rounded-lg bg-white/[0.04] text-slate-300 hover:text-white hover:bg-white/[0.08] border border-white/[0.06] transition"
                      title="Roll Key"
                    >
                      Regenerate
                    </button>
                    <button
                      onClick={() => handleRevoke(k.id)}
                      className="p-1.5 text-xs font-medium rounded-lg text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 border border-transparent hover:border-rose-500/20 transition"
                      title="Revoke Key"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>

      {/* Create Modal */}
      {isCreateOpen && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <form
            onSubmit={handleCreate}
            className="bg-[#121614] border border-white/[0.08] rounded-xl max-w-md w-full p-6 space-y-4"
          >
            <div className="flex items-center justify-between border-b border-white/[0.06] pb-3">
              <h3 className="font-semibold text-sm text-white">Generate Developer API Key</h3>
              <button
                type="button"
                onClick={() => setIsCreateOpen(false)}
                className="text-slate-400 hover:text-white text-xs font-bold"
              >
                ✕
              </button>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Key Name / Environment</label>
              <input
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Production Mobile App"
                className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Service Tier</label>
              <select
                value={tier}
                onChange={(e) => setTier(e.target.value)}
                className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
              >
                <option value="free" className="bg-[#121614] text-white">Free (10 RPS, 50,000 ETB/day limit, 1,000 req/mo)</option>
                <option value="pro" className="bg-[#121614] text-white">Pro (100 RPS, 150,000 ETB/day limit, 50,000 req/mo)</option>
                <option value="enterprise" className="bg-[#121614] text-white">Enterprise (500 RPS, Unlimited volume & quota)</option>
              </select>
            </div>

            <div className="pt-3 border-t border-white/[0.06] flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setIsCreateOpen(false)}
                className="px-4 py-2 rounded-lg text-xs font-medium text-slate-400 hover:text-white transition"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creating}
                className="px-4 py-2 rounded-lg text-xs font-semibold bg-[#00D26A] text-black hover:bg-[#00ba5e] disabled:opacity-50 transition"
              >
                {creating ? 'Generating...' : 'Create Key'}
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
