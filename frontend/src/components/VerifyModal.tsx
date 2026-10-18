import React, { useState } from 'react';
import {
  X,
  ShieldCheck,
  CheckCircle2,
  AlertCircle,
  Copy,
  Check,
  ExternalLink,
  Lock,
  ArrowRight,
  Sparkles,
} from 'lucide-react';
import { api } from '../api/client';
import type { VerificationResult } from '../types';

interface VerifyModalProps {
  isOpen: boolean;
  onClose: () => void;
  onVerified?: () => void;
}

export const VerifyModal: React.FC<VerifyModalProps> = ({ isOpen, onClose, onVerified }) => {
  const [provider, setProvider] = useState('cbe');
  const [transactionId, setTransactionId] = useState('');
  const [expectedAmount, setExpectedAmount] = useState('');
  const [referenceId, setReferenceId] = useState('');
  const [claim, setClaim] = useState(false);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<VerificationResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  if (!isOpen) return null;

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!transactionId.trim()) return;

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const res = await api.verifyPayment({
        provider,
        transactionId: transactionId.trim(),
        expectedAmount: expectedAmount ? parseFloat(expectedAmount) : undefined,
        referenceId: referenceId.trim() || undefined,
        autoClaim: claim,
      });
      setResult(res);
      if (res.verified && onVerified) {
        onVerified();
      }
    } catch (err: any) {
      setError(err.message || 'Verification request failed');
    } finally {
      setLoading(false);
    }
  };

  const copyResultJSON = () => {
    if (result) {
      navigator.clipboard.writeText(JSON.stringify(result, null, 2));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in duration-150">
      <div className="relative w-full max-w-xl bg-[#121614] border border-white/[0.08] rounded-2xl overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="px-6 py-4 border-b border-white/[0.06] flex items-center justify-between bg-black/20">
          <div className="flex items-center gap-2.5">
            <div className="h-8 w-8 rounded-lg bg-[#00D26A]/10 border border-[#00D26A]/20 flex items-center justify-center text-[#00D26A]">
              <ShieldCheck className="w-4 h-4" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-white tracking-tight">Instant Payment Verification</h3>
              <p className="text-[11px] text-slate-400">Verify and optionally claim any Ethiopian bank receipt slip</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-white/[0.04] transition"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="p-6 overflow-y-auto space-y-5">
          <form onSubmit={handleVerify} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1">Payment Provider Rail</label>
                <select
                  value={provider}
                  onChange={(e) => setProvider(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A] font-medium"
                >
                  <option value="cbe" className="bg-[#121614]">Commercial Bank of Ethiopia (CBE)</option>
                  <option value="telebirr" className="bg-[#121614]">Telebirr (Ethio Telecom)</option>
                  <option value="boa" className="bg-[#121614]">Bank of Abyssinia (BOA)</option>
                  <option value="dashen" className="bg-[#121614]">Dashen Bank</option>
                  <option value="awash" className="bg-[#121614]">Awash Bank (Awash Birr)</option>
                  <option value="mpesa" className="bg-[#121614]">M-Pesa (Safaricom Ethiopia)</option>
                  <option value="mock" className="bg-[#121614]">Mock Sandbox Engine</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1">
                  Expected Amount (ETB) <span className="text-slate-500 font-normal">Optional</span>
                </label>
                <input
                  type="number"
                  step="0.01"
                  value={expectedAmount}
                  onChange={(e) => setExpectedAmount(e.target.value)}
                  placeholder="e.g. 500.00"
                  className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A] font-mono"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-300 mb-1">
                Transaction ID or Bank Receipt URL
              </label>
              <input
                type="text"
                required
                value={transactionId}
                onChange={(e) => setTransactionId(e.target.value)}
                placeholder="e.g. FT262036S1GH10104, 387ETAP2522000WK, or receipt link"
                className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A] font-mono"
              />
              <p className="text-[10px] text-slate-500 mt-1">
                Accepts raw transaction reference codes, query params, or complete bank slip URLs.
              </p>
            </div>

            <div className="pt-2 border-t border-white/[0.06] flex items-center justify-between">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={claim}
                  onChange={(e) => setClaim(e.target.checked)}
                  className="rounded bg-black/40 border-white/[0.08] text-[#00D26A] focus:ring-0 focus:ring-offset-0 h-4 w-4"
                />
                <span className="text-xs text-slate-300 font-medium">Claim immediately for order/customer</span>
              </label>

              {claim && (
                <input
                  type="text"
                  value={referenceId}
                  onChange={(e) => setReferenceId(e.target.value)}
                  placeholder="Order / Customer ID (e.g. ord_9821)"
                  className="w-48 px-2.5 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A] font-mono"
                />
              )}
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 rounded-lg bg-[#00D26A] hover:bg-[#00ba5e] text-black font-semibold text-xs transition disabled:opacity-50 flex items-center justify-center gap-2"
            >
              {loading ? (
                'Querying Banking Network...'
              ) : (
                <>
                  <Sparkles className="w-4 h-4" /> Run Live Verification
                </>
              )}
            </button>
          </form>

          {/* Error Banner */}
          {error && (
            <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center gap-2.5">
              <AlertCircle className="w-4 h-4 shrink-0 text-rose-400" />
              <span>{error}</span>
            </div>
          )}

          {/* Verification Result Card */}
          {result && (
            <div
              className={`p-4 rounded-xl border ${
                result.verified
                  ? 'bg-[#00D26A]/5 border-[#00D26A]/20 text-emerald-200'
                  : 'bg-rose-500/5 border-rose-500/20 text-rose-200'
              } space-y-3 animate-in fade-in`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {result.verified ? (
                    <CheckCircle2 className="w-5 h-5 text-[#00D26A]" />
                  ) : (
                    <AlertCircle className="w-5 h-5 text-rose-400" />
                  )}
                  <span className="font-bold text-sm text-white">
                    {result.verified ? 'Payment Verified Successfully' : 'Verification Unsuccessful'}
                  </span>
                </div>
                <button
                  onClick={copyResultJSON}
                  className="px-2.5 py-1 rounded-lg bg-white/[0.04] hover:bg-white/[0.08] text-[10px] text-slate-300 hover:text-white flex items-center gap-1 font-mono border border-white/[0.06] transition"
                >
                  {copied ? <Check className="w-3 h-3 text-[#00D26A]" /> : <Copy className="w-3 h-3" />}
                  {copied ? 'Copied' : 'Copy JSON'}
                </button>
              </div>

              {result.verified ? (
                <div className="grid grid-cols-2 gap-2 text-xs bg-black/40 p-3 rounded-lg border border-white/[0.04] font-mono">
                  <div>
                    <span className="text-slate-500 block text-[10px] uppercase font-sans">Verified Amount</span>
                    <span className="text-white font-bold text-sm">
                      {result.amount?.toLocaleString()} {result.currency || 'ETB'}
                    </span>
                  </div>
                  <div>
                    <span className="text-slate-500 block text-[10px] uppercase font-sans">Provider Rail</span>
                    <span className="text-[#00D26A] uppercase font-semibold">{result.providerKey}</span>
                  </div>
                  <div>
                    <span className="text-slate-500 block text-[10px] uppercase font-sans">Payer</span>
                    <span className="text-slate-300 truncate block">{result.payerIdentifier || 'N/A'}</span>
                  </div>
                  <div>
                    <span className="text-slate-500 block text-[10px] uppercase font-sans">Receiver / Merchant</span>
                    <span className="text-slate-300 truncate block">{result.receiverIdentifier || 'N/A'}</span>
                  </div>
                  <div>
                    <span className="text-slate-500 block text-[10px] uppercase font-sans">Claim Status</span>
                    <span className={result.claimed ? 'text-[#00D26A] font-semibold' : 'text-amber-400'}>
                      {result.claimed ? 'Claimed' : 'Unclaimed'}
                    </span>
                  </div>
                  <div>
                    <span className="text-slate-500 block text-[10px] uppercase font-sans">Timestamp</span>
                    <span className="text-slate-400 text-[11px] truncate block">{result.timestamp || 'N/A'}</span>
                  </div>
                </div>
              ) : (
                <div className="text-xs text-rose-300 bg-black/40 p-3 rounded-lg border border-rose-500/20">
                  <span className="font-semibold block">{result.errorCode || 'VERIFICATION_FAILED'}</span>
                  <p className="text-[11px] text-slate-400 mt-1">{result.errorMessage || 'Transaction not recognized.'}</p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
