import React, { useState } from 'react';
import {
  X,
  CreditCard,
  CheckCircle2,
  AlertCircle,
  Copy,
  Check,
  Calendar,
  User,
  Building,
  Hash,
  Shield,
  FileCode,
  ArrowRight,
  ExternalLink,
} from 'lucide-react';
import type { Payment } from '../types';
import { api } from '../api/client';

interface TransactionDrawerProps {
  payment: Payment | null;
  onClose: () => void;
  onUpdated?: () => void;
}

export const TransactionDrawer: React.FC<TransactionDrawerProps> = ({ payment, onClose, onUpdated }) => {
  const [activeTab, setActiveTab] = useState<'details' | 'json'>('details');
  const [copied, setCopied] = useState(false);
  const [reconciling, setReconciling] = useState(false);
  const [reconcileSuccess, setReconcileSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!payment) return null;

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleReconcile = async () => {
    setReconciling(true);
    setError(null);
    try {
      await api.reconcileTransaction(payment.id);
      setReconcileSuccess(true);
      if (onUpdated) onUpdated();
    } catch (err: any) {
      setError(err.message || 'Reconciliation failed');
    } finally {
      setReconciling(false);
    }
  };

  const getProviderColor = (prov: string) => {
    switch (prov.toLowerCase()) {
      case 'cbe':
        return 'text-purple-400 bg-purple-500/10 border-purple-500/30';
      case 'telebirr':
        return 'text-emerald-400 bg-emerald-500/10 border-emerald-500/30';
      case 'boa':
        return 'text-amber-400 bg-amber-500/10 border-amber-500/30';
      case 'dashen':
        return 'text-blue-400 bg-blue-500/10 border-blue-500/30';
      case 'awash':
        return 'text-orange-400 bg-orange-500/10 border-orange-500/30';
      case 'mpesa':
        return 'text-rose-400 bg-rose-500/10 border-rose-500/30';
      default:
        return 'text-slate-400 bg-slate-500/10 border-slate-500/30';
    }
  };

  return (
    <div className="fixed inset-0 z-50 overflow-hidden bg-black/70 backdrop-blur-sm animate-in fade-in">
      <div className="absolute inset-y-0 right-0 max-w-full flex pl-10">
        <div className="w-screen max-w-lg bg-[#121614] border-l border-white/[0.08] flex flex-col">
          {/* Header */}
          <div className="p-6 border-b border-white/[0.06] bg-black/20 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-[#00D26A]/10 border border-[#00D26A]/20 flex items-center justify-center text-[#00D26A]">
                <CreditCard className="w-5 h-5" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold text-white tracking-tight">Transaction Details</span>
                  <span
                    className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border ${getProviderColor(
                      payment.providerKey
                    )}`}
                  >
                    {payment.providerKey}
                  </span>
                </div>
                <div className="flex items-center gap-1.5 mt-0.5">
                  <span className="text-[11px] font-mono text-slate-400 truncate max-w-[240px]">
                    {payment.transactionId}
                  </span>
                  <button
                    onClick={() => handleCopy(payment.transactionId)}
                    title="Copy Transaction ID"
                    className="text-slate-500 hover:text-slate-300 transition"
                  >
                    {copied ? <Check className="w-3 h-3 text-[#00D26A]" /> : <Copy className="w-3 h-3 text-slate-400" />}
                  </button>
                </div>
              </div>
            </div>

            <button
              onClick={onClose}
              className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-white/[0.04] transition"
            >
              <X className="w-4 h-4" />
            </button>
          </div>

          {/* Sub Navigation */}
          <div className="flex border-b border-white/[0.06] px-6 bg-black/20">
            <button
              onClick={() => setActiveTab('details')}
              className={`py-3 text-xs font-semibold border-b-2 mr-6 transition ${
                activeTab === 'details'
                  ? 'border-[#00D26A] text-white'
                  : 'border-transparent text-slate-400 hover:text-slate-200'
              }`}
            >
              Receipt Slip
            </button>
            <button
              onClick={() => setActiveTab('json')}
              className={`py-3 text-xs font-semibold border-b-2 transition flex items-center gap-1.5 ${
                activeTab === 'json'
                  ? 'border-[#00D26A] text-white'
                  : 'border-transparent text-slate-400 hover:text-slate-200'
              }`}
            >
              <FileCode className="w-3.5 h-3.5" /> Raw Telemetry JSON
            </button>
          </div>

          {/* Content Area */}
          <div className="flex-1 overflow-y-auto p-6 space-y-6">
            {activeTab === 'details' ? (
              <>
                {/* Financial Overview Card */}
                <div className="p-4 rounded-xl bg-black/40 border border-white/[0.04] space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-slate-400">Total Settled Amount</span>
                    <span
                      className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider border ${
                        payment.status === 'VERIFIED'
                          ? 'bg-[#00D26A]/10 text-[#00D26A] border-[#00D26A]/20'
                          : payment.status === 'CLAIMED'
                          ? 'bg-blue-500/10 text-blue-400 border-blue-500/20'
                          : 'bg-rose-500/10 text-rose-400 border-rose-500/20'
                      }`}
                    >
                      {payment.status}
                    </span>
                  </div>
                  <div className="text-2xl font-black text-white font-mono tracking-tight">
                    {payment.amount?.toLocaleString('en-US', { minimumFractionDigits: 2 })} {payment.currency || 'ETB'}
                  </div>
                  <div className="text-[11px] text-slate-400 flex items-center gap-1.5">
                    <Calendar className="w-3.5 h-3.5 text-slate-500" />
                    <span>Verified on {payment.verifiedAt ? new Date(payment.verifiedAt).toLocaleString() : new Date(payment.createdAt).toLocaleString()}</span>
                  </div>
                </div>

                {/* Parties Information */}
                <div className="space-y-3">
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400">Parties & Routing</h4>
                  <div className="p-4 rounded-xl bg-black/40 border border-white/[0.04] space-y-3 text-xs">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-2 text-slate-400">
                        <User className="w-4 h-4 text-slate-500" />
                        <span>Payer Name / ID</span>
                      </div>
                      <span className="font-semibold text-white text-right max-w-[220px] truncate">
                        {payment.payerIdentifier || 'Not Provided'}
                      </span>
                    </div>

                    <div className="border-t border-white/[0.04] pt-2 flex items-start justify-between">
                      <div className="flex items-center gap-2 text-slate-400">
                        <Building className="w-4 h-4 text-slate-500" />
                        <span>Receiver / Merchant</span>
                      </div>
                      <span className="font-semibold text-white text-right max-w-[220px] truncate">
                        {payment.receiverIdentifier || 'Merchant Account'}
                      </span>
                    </div>

                    <div className="border-t border-white/[0.04] pt-2 flex items-start justify-between">
                      <div className="flex items-center gap-2 text-slate-400">
                        <Hash className="w-4 h-4 text-slate-500" />
                        <span>Claimed Reference</span>
                      </div>
                      <span className="font-mono text-[#00D26A] text-right">
                        {payment.referenceId || (payment.claimed ? 'Claimed (Default)' : 'None (Unclaimed)')}
                      </span>
                    </div>

                    {payment.apiKeyName && (
                      <div className="border-t border-white/[0.04] pt-2 flex items-start justify-between">
                        <div className="flex items-center gap-2 text-slate-400">
                          <Shield className="w-4 h-4 text-slate-500" />
                          <span>Caller API Key</span>
                        </div>
                        <span className="font-mono text-slate-300 text-right">
                          {payment.apiKeyName} ({payment.apiKeyPrefix}...)
                        </span>
                      </div>
                    )}
                  </div>
                </div>

                {/* Error Banner if applicable */}
                {payment.errorMessage && (
                  <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center gap-2.5">
                    <AlertCircle className="w-4 h-4 shrink-0 text-rose-400" />
                    <div>
                      <div className="font-bold uppercase tracking-wider text-[10px] text-rose-400">
                        {payment.errorCode || 'VERIFICATION_ERROR'}
                      </div>
                      <div>{payment.errorMessage}</div>
                    </div>
                  </div>
                )}

                {/* Admin Actions */}
                <div className="space-y-3 pt-2">
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400">Reconciliation Action</h4>
                  {reconcileSuccess ? (
                    <div className="p-3 rounded-lg bg-[#00D26A]/10 border border-[#00D26A]/20 text-[#00D26A] text-xs flex items-center gap-2">
                      <CheckCircle2 className="w-4 h-4" />
                      <span>Transaction reconciled and verified successfully!</span>
                    </div>
                  ) : (
                    <button
                      onClick={handleReconcile}
                      disabled={reconciling}
                      className="w-full py-2.5 rounded-lg bg-white/[0.04] hover:bg-white/[0.08] text-slate-200 font-semibold text-xs transition border border-white/[0.06] flex items-center justify-center gap-2 disabled:opacity-50"
                    >
                      {reconciling ? 'Re-verifying with Bank...' : 'Re-verify with Upstream Bank (Reconcile)'}
                    </button>
                  )}
                  {error && <p className="text-xs text-rose-400 mt-1">{error}</p>}
                </div>
              </>
            ) : (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs text-slate-400">Full Record Payload</span>
                  <button
                    onClick={() => handleCopy(JSON.stringify(payment, null, 2))}
                    className="px-2.5 py-1 rounded-lg bg-white/[0.04] hover:bg-white/[0.08] text-[10px] text-slate-300 hover:text-white flex items-center gap-1 font-mono border border-white/[0.06] transition"
                  >
                    {copied ? <Check className="w-3 h-3 text-[#00D26A]" /> : <Copy className="w-3 h-3" />}
                    {copied ? 'Copied' : 'Copy'}
                  </button>
                </div>
                <pre className="p-4 rounded-xl bg-black/40 border border-white/[0.04] text-[11px] font-mono text-[#00D26A] overflow-x-auto max-h-[600px] leading-relaxed">
                  {JSON.stringify(payment, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
