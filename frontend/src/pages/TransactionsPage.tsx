import React, { useEffect, useState } from 'react';
import {
  CreditCard,
  Search,
  Filter,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Eye,
  ShieldCheck,
  AlertCircle,
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  Plus,
  Sparkles,
  Copy,
  Check,
  ArrowUpRight,
} from 'lucide-react';
import type { Payment, User } from '../types';
import { api } from '../api/client';

interface TransactionsPageProps {
  user: User | null;
  onSelectPayment?: (payment: Payment) => void;
  onOpenVerify?: () => void;
}

const PROVIDERS = [
  { key: '', label: 'All Rails', badge: 'bg-slate-800 text-slate-300' },
  { key: 'cbe', label: 'CBE', badge: 'bg-purple-500/10 text-purple-400 border border-purple-500/20' },
  { key: 'telebirr', label: 'Telebirr', badge: 'bg-sky-500/10 text-sky-400 border border-sky-500/20' },
  { key: 'boa', label: 'Abyssinia', badge: 'bg-amber-500/10 text-amber-400 border border-amber-500/20' },
  { key: 'dashen', label: 'Dashen', badge: 'bg-blue-500/10 text-blue-400 border border-blue-500/20' },
  { key: 'awash', label: 'Awash', badge: 'bg-orange-500/10 text-orange-400 border border-orange-500/20' },
  { key: 'mpesa', label: 'M-Pesa', badge: 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' },
];

export const TransactionsPage: React.FC<TransactionsPageProps> = ({
  user,
  onSelectPayment,
  onOpenVerify,
}) => {
  const [transactions, setTransactions] = useState<Payment[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [provider, setProvider] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(0);
  const [limit] = useState(25);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  // Local modal fallback if onSelectPayment is not provided
  const [selectedTx, setSelectedTx] = useState<Payment | null>(null);
  const [reconciling, setReconciling] = useState(false);
  const [reconcileMsg, setReconcileMsg] = useState<string | null>(null);

  const fetchTransactions = async () => {
    setLoading(true);
    try {
      const res = await api.listTransactions({
        search: search.trim() || undefined,
        provider: provider || undefined,
        status: status || undefined,
        limit,
        offset: page * limit,
      });
      setTransactions(res.items || []);
      setTotal(res.total || 0);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTransactions();
  }, [provider, status, page]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(0);
    fetchTransactions();
  };

  const handleRowClick = (tx: Payment) => {
    if (onSelectPayment) {
      onSelectPayment(tx);
    } else {
      setSelectedTx(tx);
      setReconcileMsg(null);
    }
  };

  const handleCopy = (id: string, text: string, e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 1800);
  };

  const handleReconcile = async () => {
    if (!selectedTx) return;
    setReconciling(true);
    setReconcileMsg(null);
    try {
      const updated = await api.reconcileTransaction(selectedTx.id);
      setSelectedTx(updated);
      setReconcileMsg('Transaction successfully reconciled with provider');
      fetchTransactions();
    } catch (err: any) {
      setReconcileMsg(`Reconciliation failed: ${err.message}`);
    } finally {
      setReconciling(false);
    }
  };

  const totalPages = Math.ceil(total / limit);

  return (
    <div className="space-y-6">
      {/* Header with Title & Quick Verify Action */}
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold text-white">Transactions</h1>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            onClick={() => fetchTransactions()}
            className="p-2 rounded-lg bg-white/[0.04] text-slate-400 hover:text-white border border-white/[0.06] hover:border-white/[0.12] transition"
            title="Refresh"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
          {onOpenVerify && (
            <button
              onClick={onOpenVerify}
              className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-[#00D26A] hover:bg-[#00ba5e] text-black text-xs font-semibold transition"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Verify</span>
              <span className="hidden sm:inline-block ml-1 px-1.5 py-0.5 rounded bg-black/20 text-[10px] text-black/80 font-mono">
                ⌘K
              </span>
            </button>
          )}
        </div>
      </div>

      {/* Filter Toolbar: Search Bar + Provider Pills + Status */}
      <div className="p-3.5 rounded-xl bg-[#121614] border border-white/[0.06] space-y-3">
        <div className="flex flex-col md:flex-row gap-2.5 items-center justify-between">
          <form onSubmit={handleSearchSubmit} className="relative w-full md:w-80">
            <Search className="w-3.5 h-3.5 text-slate-500 absolute left-3 top-2.5" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search reference, tx ID, payer..."
              className="w-full pl-9 pr-3 py-1.5 rounded-lg bg-black/50 border border-white/[0.08] text-xs text-white placeholder-slate-500 focus:outline-none focus:border-[#00D26A]"
            />
            {search && (
              <button
                type="button"
                onClick={() => {
                  setSearch('');
                  setPage(0);
                }}
                className="absolute right-2.5 top-2 text-[10px] text-slate-500 hover:text-slate-300"
              >
                Clear
              </button>
            )}
          </form>

          <div className="flex items-center gap-2 w-full md:w-auto">
            <select
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(0);
              }}
              className="px-3 py-1.5 rounded-lg bg-black/50 border border-white/[0.08] text-xs text-slate-300 focus:outline-none focus:border-[#00D26A]"
            >
              <option value="">All Statuses</option>
              <option value="VERIFIED">VERIFIED</option>
              <option value="CLAIMED">CLAIMED</option>
              <option value="FAILED">FAILED</option>
            </select>
          </div>
        </div>

        {/* Provider Rail Quick-Filter Pills */}
        <div className="flex items-center gap-1 overflow-x-auto pt-0.5 pb-0.5 scrollbar-thin">
          <span className="text-[11px] text-slate-500 font-medium mr-1 shrink-0 flex items-center gap-1">
            <Filter className="w-3 h-3" />
            Rail:
          </span>
          {PROVIDERS.map((p) => {
            const active = provider === p.key;
            return (
              <button
                key={p.key || 'all'}
                onClick={() => {
                  setProvider(p.key);
                  setPage(0);
                }}
                className={`px-2.5 py-1 rounded-md text-xs font-medium whitespace-nowrap transition ${
                  active
                    ? 'bg-[#00D26A] text-black font-semibold'
                    : 'bg-white/[0.04] text-slate-400 hover:text-slate-200 hover:bg-white/[0.08]'
                }`}
              >
                {p.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Transactions Data Table */}
      <div className="rounded-xl bg-[#121614] border border-white/[0.06] overflow-hidden">
        {loading ? (
          <div className="py-20 text-center space-y-2">
            <div className="inline-block h-5 w-5 rounded-full border-2 border-[#00D26A] border-t-transparent animate-spin" />
            <div className="text-xs text-slate-400">Querying settlement ledger...</div>
          </div>
        ) : transactions.length === 0 ? (
          <div className="py-16 text-center space-y-2">
            <CreditCard className="w-8 h-8 text-slate-600 mx-auto" />
            <div className="text-sm font-semibold text-slate-300">No transactions found</div>
            <p className="text-xs text-slate-500 max-w-sm mx-auto">
              No verification records matched the specified filters. Try resetting the search or verify a new receipt slip.
            </p>
            {onOpenVerify && (
              <button
                onClick={onOpenVerify}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-[#00D26A] text-black text-xs font-semibold hover:bg-[#25F28A] transition mt-2"
              >
                <Plus className="w-3.5 h-3.5" />
                Verify a Slip Now
              </button>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-white/[0.06] text-slate-500 font-medium uppercase tracking-wider text-[10px]">
                  <th className="py-2.5 px-3.5">Provider / Rail</th>
                  <th className="py-2.5 px-3.5">Transaction Reference</th>
                  <th className="py-2.5 px-3.5">Settled Amount</th>
                  <th className="py-2.5 px-3.5">Counterparties</th>
                  <th className="py-2.5 px-3.5">Claim State</th>
                  <th className="py-2.5 px-3.5">Status</th>
                  <th className="py-2.5 px-3.5">Verified At</th>
                  <th className="py-2.5 px-3.5 text-right">Inspect</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04]">
                {transactions.map((tx) => {
                  const isClaimed = tx.claimed || tx.status === 'CLAIMED';
                  return (
                    <tr
                      key={tx.id}
                      onClick={() => handleRowClick(tx)}
                      className="hover:bg-white/[0.02] transition cursor-pointer group"
                    >
                      {/* Provider Rail */}
                      <td className="py-3 px-3.5">
                        <span className="font-semibold uppercase text-[10px] px-2 py-0.5 rounded bg-white/[0.04] text-slate-300">
                          {tx.providerKey}
                        </span>
                      </td>

                      {/* Transaction ID & Copy button */}
                      <td className="py-3 px-3.5 font-mono font-medium text-slate-200">
                        <div className="flex items-center gap-1.5">
                          <span>{tx.transactionId}</span>
                          <button
                            onClick={(e) => handleCopy(tx.id, tx.transactionId, e)}
                            className="opacity-0 group-hover:opacity-100 p-1 text-slate-500 hover:text-white transition"
                            title="Copy Transaction ID"
                          >
                            {copiedId === tx.id ? (
                              <Check className="w-3 h-3 text-[#00D26A]" />
                            ) : (
                              <Copy className="w-3 h-3" />
                            )}
                          </button>
                        </div>
                        {tx.referenceId && tx.referenceId !== tx.transactionId && (
                          <div className="text-[10px] text-slate-500 font-sans">
                            Ref: {tx.referenceId}
                          </div>
                        )}
                      </td>

                      {/* Amount */}
                      <td className="py-3 px-3.5">
                        <div className="font-bold text-white text-xs font-mono">
                          {tx.amount.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                          <span className="text-slate-400 text-[10px] font-normal ml-1 font-sans">
                            {tx.currency || 'ETB'}
                          </span>
                        </div>
                      </td>

                      {/* Counterparties */}
                      <td className="py-3 px-3.5 text-slate-300">
                        {tx.payerIdentifier ? (
                          <div className="text-slate-200 font-medium truncate max-w-[180px]">
                            {tx.payerIdentifier}
                          </div>
                        ) : (
                          <span className="text-slate-500 italic">Unknown Payer</span>
                        )}
                        {tx.receiverIdentifier && (
                          <div className="text-[10px] text-slate-500 truncate max-w-[180px]">
                            → {tx.receiverIdentifier}
                          </div>
                        )}
                      </td>

                      {/* Claim State */}
                      <td className="py-3 px-3.5">
                        {isClaimed ? (
                          <div className="inline-flex items-center gap-1 text-[#00D26A] font-medium text-[11px]">
                            <CheckCircle2 className="w-3.5 h-3.5" />
                            Claimed
                          </div>
                        ) : (
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-500/10 text-amber-400">
                            Available
                          </span>
                        )}
                      </td>

                      {/* Status */}
                      <td className="py-3 px-3.5">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-semibold ${
                            tx.status === 'VERIFIED' || tx.status === 'CLAIMED'
                              ? 'bg-[#00D26A]/10 text-[#00D26A]'
                              : 'bg-rose-500/10 text-rose-400'
                          }`}
                        >
                          {tx.status}
                        </span>
                      </td>

                      {/* Verified At */}
                      <td className="py-3 px-3.5 text-slate-500 font-mono text-[11px]">
                        {new Date(tx.createdAt).toLocaleDateString()} {new Date(tx.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </td>

                      {/* Action */}
                      <td className="py-3 px-3.5 text-right">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleRowClick(tx);
                          }}
                          className="px-2 py-1 rounded bg-white/[0.04] text-slate-300 hover:text-white hover:bg-white/[0.08] text-xs font-medium transition inline-flex items-center gap-1"
                        >
                          Inspect
                          <ArrowUpRight className="w-3 h-3 text-slate-400" />
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination Bar */}
        <div className="p-4 border-t border-white/[0.06] flex items-center justify-between text-xs text-slate-400 bg-black/20">
          <span>
            Showing {transactions.length} of {total.toLocaleString()} records
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
              className="p-1.5 rounded-lg bg-white/[0.04] text-slate-300 disabled:opacity-30 hover:bg-white/[0.08] transition"
              title="Previous Page"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <span className="font-mono">
              Page {page + 1} of {Math.max(1, totalPages)}
            </span>
            <button
              onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
              disabled={page >= totalPages - 1}
              className="p-1.5 rounded-lg bg-white/[0.04] text-slate-300 disabled:opacity-30 hover:bg-white/[0.08] transition"
              title="Next Page"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Fallback Transaction Details Modal (when onSelectPayment is not injected) */}
      {selectedTx && !onSelectPayment && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-[#121614] border border-white/[0.08] rounded-xl max-w-lg w-full p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-white/[0.06] pb-3">
              <h3 className="font-semibold text-sm text-white flex items-center gap-2">
                <CreditCard className="w-4 h-4 text-[#00D26A]" />
                Transaction Record Details
              </h3>
              <button
                onClick={() => setSelectedTx(null)}
                className="text-slate-400 hover:text-white text-xs font-bold"
              >
                ✕
              </button>
            </div>

            {reconcileMsg && (
              <div className="p-3 rounded-lg bg-[#00D26A]/10 border border-[#00D26A]/20 text-[#00D26A] text-xs">
                {reconcileMsg}
              </div>
            )}

            <div className="space-y-2 text-xs">
              <div className="flex justify-between py-1 border-b border-white/[0.04]">
                <span className="text-slate-400">ID:</span>
                <span className="font-mono text-slate-200">{selectedTx.id}</span>
              </div>
              <div className="flex justify-between py-1 border-b border-white/[0.04]">
                <span className="text-slate-400">Provider:</span>
                <span className="font-bold uppercase text-[#00D26A]">{selectedTx.providerKey}</span>
              </div>
              <div className="flex justify-between py-1 border-b border-white/[0.04]">
                <span className="text-slate-400">Transaction ID:</span>
                <span className="font-mono text-slate-200 font-semibold">{selectedTx.transactionId}</span>
              </div>
              <div className="flex justify-between py-1 border-b border-white/[0.04]">
                <span className="text-slate-400">Amount:</span>
                <span className="font-bold text-white">
                  {selectedTx.amount.toLocaleString()} {selectedTx.currency}
                </span>
              </div>
              <div className="flex justify-between py-1 border-b border-white/[0.04]">
                <span className="text-slate-400">Status:</span>
                <span className="font-bold text-emerald-400">{selectedTx.status}</span>
              </div>
              <div className="flex justify-between py-1 border-b border-white/[0.04]">
                <span className="text-slate-400">Claimed:</span>
                <span>{selectedTx.claimed ? `Yes (${selectedTx.claimedBy || 'system'})` : 'No'}</span>
              </div>
            </div>

            <div className="pt-3 border-t border-white/[0.06] flex items-center justify-between gap-3">
              {user?.role === 'admin' && (
                <button
                  onClick={handleReconcile}
                  disabled={reconciling}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/[0.05] text-slate-200 border border-white/[0.08] text-xs font-semibold hover:bg-white/[0.1] transition disabled:opacity-50"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${reconciling ? 'animate-spin' : ''}`} />
                  Reconcile with Provider
                </button>
              )}
              <button
                onClick={() => setSelectedTx(null)}
                className="ml-auto px-4 py-2 rounded-lg bg-[#00D26A] text-black hover:bg-[#00ba5e] text-xs font-semibold transition"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
