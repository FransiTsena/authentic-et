import React from 'react';
import { FileText, ExternalLink, Code2, ShieldAlert, Key, CheckCircle2, ArrowLeft } from 'lucide-react';
import { BrandLogo } from '../components/BrandLogo';

interface DocsPageProps {
  isPublic?: boolean;
  onBack?: () => void;
  onNavigateToHealth?: () => void;
  onOpenSignIn?: () => void;
}

export const DocsPage: React.FC<DocsPageProps> = ({
  isPublic = false,
  onBack,
  onNavigateToHealth,
  onOpenSignIn,
}) => {
  const content = (
    <div className="space-y-8 max-w-4xl mx-auto">
      <div>
        <h1 className="text-lg font-semibold text-white">Documentation</h1>
      </div>

      <div className="flex items-center gap-3">
        <a
          href="/api/v1/openapi.json"
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/[0.04] border border-white/[0.06] text-xs font-semibold text-slate-200 hover:text-white hover:bg-white/[0.08] transition"
        >
          <ExternalLink className="w-3.5 h-3.5 text-[#00D26A]" />
          OpenAPI 3.0 Specification (JSON)
        </a>
      </div>

      {/* Quickstart Authentication */}
      <div className="p-6 rounded-xl bg-[#121614] border border-white/[0.06] space-y-3">
        <h2 className="text-sm font-bold text-white flex items-center gap-2">
          <Key className="w-4 h-4 text-[#00D26A]" />
          Authentication & Headers
        </h2>
        <p className="text-xs text-slate-300 leading-relaxed">
          Public verification endpoints can be used anonymously for testing or authenticated with a developer API key via the <code className="text-[#00D26A] bg-black/40 px-1.5 py-0.5 rounded border border-white/[0.06] font-mono">X-API-KEY</code> header.
        </p>
        <div className="p-3 bg-black/40 rounded-lg border border-white/[0.04] text-xs font-mono text-slate-300">
          X-API-KEY: sk_pv_a1b2c3d4e5f6...
        </div>
      </div>

      {/* Endpoint: Verify Transaction */}
      <div className="p-6 rounded-xl bg-[#121614] border border-white/[0.06] space-y-4">
        <div className="flex items-center gap-2">
          <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-[#00D26A]/10 text-[#00D26A] border border-[#00D26A]/20">
            POST
          </span>
          <span className="font-mono text-xs font-bold text-white">/api/v1/public/payment/verify</span>
        </div>
        <p className="text-xs text-slate-400">
          Verifies a payment receipt or bank transaction ID against the specified payment rail. Validates amount tolerances and receiver merchant identity.
        </p>

        <h3 className="text-xs font-bold text-slate-300 uppercase tracking-wider pt-2">Request Body (JSON)</h3>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-white/[0.06] text-slate-400">
                <th className="py-2 px-3">Field</th>
                <th className="py-2 px-3">Type</th>
                <th className="py-2 px-3">Required</th>
                <th className="py-2 px-3">Description</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/[0.04] text-slate-300">
              <tr>
                <td className="py-2.5 px-3 font-mono text-[#00D26A] font-semibold">provider</td>
                <td className="py-2.5 px-3 font-mono text-slate-400">string</td>
                <td className="py-2.5 px-3 text-rose-400 font-semibold">Yes</td>
                <td className="py-2.5 px-3">Provider key: <code>cbe</code>, <code>telebirr</code>, <code>boa</code>, <code>dashen</code>, <code>awash</code>, <code>mpesa</code>, or <code>mock</code>.</td>
              </tr>
              <tr>
                <td className="py-2.5 px-3 font-mono text-[#00D26A] font-semibold">transactionId</td>
                <td className="py-2.5 px-3 font-mono text-slate-400">string</td>
                <td className="py-2.5 px-3 text-rose-400 font-semibold">Yes</td>
                <td className="py-2.5 px-3">Transaction reference code or full bank receipt slip URL.</td>
              </tr>
              <tr>
                <td className="py-2.5 px-3 font-mono text-[#00D26A] font-semibold">expectedAmount</td>
                <td className="py-2.5 px-3 font-mono text-slate-400">number</td>
                <td className="py-2.5 px-3 text-slate-500">No</td>
                <td className="py-2.5 px-3">Expected payment amount in ETB. Enforces -1.00 ETB / +5.00 ETB tolerance.</td>
              </tr>
              <tr>
                <td className="py-2.5 px-3 font-mono text-[#00D26A] font-semibold">autoClaim</td>
                <td className="py-2.5 px-3 font-mono text-slate-400">boolean</td>
                <td className="py-2.5 px-3 text-slate-500">No</td>
                <td className="py-2.5 px-3">If true, marks transaction as claimed immediately upon verification.</td>
              </tr>
              <tr>
                <td className="py-2.5 px-3 font-mono text-[#00D26A] font-semibold">claimedBy</td>
                <td className="py-2.5 px-3 font-mono text-slate-400">string</td>
                <td className="py-2.5 px-3 text-slate-500">No</td>
                <td className="py-2.5 px-3">Order or customer identifier claiming the transaction.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* Endpoint: Claim Transaction */}
      <div className="p-6 rounded-xl bg-[#121614] border border-white/[0.06] space-y-4">
        <div className="flex items-center gap-2">
          <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            POST
          </span>
          <span className="font-mono text-xs font-bold text-white">/api/v1/public/payment/claim</span>
        </div>
        <p className="text-xs text-slate-400">
          Atomically marks a verified transaction as claimed. Subsequent claim attempts will be rejected to prevent double spending.
        </p>
      </div>

      {/* Error Codes Reference */}
      <div className="p-6 rounded-xl bg-[#121614] border border-white/[0.06] space-y-4">
        <h2 className="text-sm font-bold text-white flex items-center gap-2">
          <ShieldAlert className="w-4 h-4 text-amber-400" />
          Standard Machine Error Codes
        </h2>
        <div className="space-y-2 text-xs">
          {[
            { code: 'TRANSACTION_NOT_FOUND', desc: 'The reference was not found on the bank/telecom ledger, or slip is invalid.' },
            { code: 'TRANSACTION_ALREADY_CLAIMED', desc: 'The transaction has already been claimed by another order or user.' },
            { code: 'AMOUNT_MISMATCH', desc: 'The amount credited does not match expected amount beyond allowed tolerance.' },
            { code: 'RECIPIENT_MISMATCH', desc: 'The credited party on the slip does not match the configured merchant account.' },
            { code: 'RATE_LIMIT_EXCEEDED', desc: 'Request rate limit exceeded. Retry after specified window or supply API key.' },
          ].map((item) => (
            <div key={item.code} className="p-3 rounded-lg bg-black/40 border border-white/[0.04] flex flex-col sm:flex-row sm:items-center justify-between gap-2">
              <code className="text-amber-400 font-mono font-bold text-[11px]">{item.code}</code>
              <span className="text-slate-400 text-[11px]">{item.desc}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );

  if (!isPublic) {
    return content;
  }

  return (
    <div className="min-h-screen bg-[#0c0f0d] text-slate-100 antialiased font-sans flex flex-col">
      {/* Public Top Header */}
      <header className="h-16 border-b border-white/[0.06] bg-[#0c0f0d]/80 backdrop-blur-md sticky top-0 z-30 px-4 sm:px-8 flex items-center justify-between">
        <div className="flex items-center gap-4">
          {onBack && (
            <>
              <button
                onClick={onBack}
                className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-white transition"
              >
                <ArrowLeft className="w-4 h-4" />
                <span>Back</span>
              </button>
              <div className="h-4 w-px bg-white/[0.08]" />
            </>
          )}
          <BrandLogo size={28} showText={true} />
        </div>

        <div className="flex items-center gap-3">
          {onNavigateToHealth && (
            <button
              onClick={onNavigateToHealth}
              className="text-xs text-slate-400 hover:text-white transition px-3 py-1.5 rounded-lg hover:bg-white/[0.04]"
            >
              System Health
            </button>
          )}
          {onOpenSignIn && (
            <button
              onClick={onOpenSignIn}
              className="px-3.5 py-1.5 rounded-lg bg-white/[0.06] hover:bg-white/[0.1] text-xs font-semibold text-white transition"
            >
              Sign In
            </button>
          )}
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 max-w-4xl w-full mx-auto p-4 sm:p-8">
        {content}
      </main>

      {/* Footer */}
      <footer className="border-t border-white/[0.06] px-8 py-4 flex items-center justify-between text-xs text-slate-500">
        <div>authentic-et</div>
        <div>Open Source</div>
      </footer>
    </div>
  );
};
