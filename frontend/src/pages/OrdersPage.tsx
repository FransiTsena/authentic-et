import React, { useEffect, useState } from 'react';
import {
  Layers,
  Plus,
  CheckCircle2,
  Clock,
  RefreshCw,
  ExternalLink,
  ShieldCheck,
} from 'lucide-react';
import type { Order } from '../types';
import { api } from '../api/client';

export const OrdersPage: React.FC = () => {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [isVerifyOpen, setIsVerifyOpen] = useState(false);
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);
  const [provider, setProvider] = useState('cbe');
  const [txId, setTxId] = useState('TEST_SUCCESS');
  const [verifying, setVerifying] = useState(false);
  const [verifyMsg, setVerifyMsg] = useState<string | null>(null);

  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [serviceType, setServiceType] = useState('vps');
  const [tier, setTier] = useState('standard');
  const [amount, setAmount] = useState('500');

  const fetchOrders = async () => {
    setLoading(true);
    try {
      const res = await api.listOrders();
      setOrders(res.items || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchOrders();
  }, []);

  const handleCreateOrder = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.createOrder({
        serviceType,
        tier,
        amount: parseFloat(amount) || 500,
        billingPeriod: 1,
      });
      setIsCreateOpen(false);
      fetchOrders();
    } catch (err: any) {
      alert(`Failed to create order: ${err.message}`);
    }
  };

  const handleVerifyOrderPayment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedOrder) return;
    setVerifying(true);
    setVerifyMsg(null);
    try {
      const res = await api.verifyOrder(selectedOrder.id, provider, txId.trim());
      setVerifyMsg(res.message || 'Payment verified and service provisioning queued.');
      fetchOrders();
    } catch (err: any) {
      setVerifyMsg(`Error: ${err.message}`);
    } finally {
      setVerifying(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold text-white">Orders</h1>
        </div>
        <button
          onClick={() => setIsCreateOpen(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg bg-[#00D26A] text-black hover:bg-[#00ba5e] transition"
        >
          <Plus className="w-3.5 h-3.5" />
          Create Order
        </button>
      </div>

      <div className="rounded-xl bg-[#121614] border border-white/[0.06] overflow-hidden">
        {loading ? (
          <div className="py-20 text-center text-xs text-slate-500">Loading orders...</div>
        ) : orders.length === 0 ? (
          <div className="py-20 text-center text-xs text-slate-500">
            No service orders found. Create a test order above to simulate checkout!
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-white/[0.06] text-slate-400 font-medium bg-black/20">
                  <th className="py-3 px-4">Order ID</th>
                  <th className="py-3 px-4">Service Type</th>
                  <th className="py-3 px-4">Tier</th>
                  <th className="py-3 px-4">Amount</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4">Payment Ref</th>
                  <th className="py-3 px-4">Created</th>
                  <th className="py-3 px-4 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04]">
                {orders.map((o) => (
                  <tr key={o.id} className="hover:bg-white/[0.02] transition">
                    <td className="py-3 px-4 font-mono font-semibold text-slate-200">{o.id}</td>
                    <td className="py-3 px-4 uppercase text-[#00D26A] font-medium">{o.serviceType}</td>
                    <td className="py-3 px-4 text-slate-300 capitalize">{o.tier}</td>
                    <td className="py-3 px-4 font-bold text-slate-100">
                      {o.amount.toLocaleString()} {o.currency}
                    </td>
                    <td className="py-3 px-4">
                      <span
                        className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold border ${
                          o.status === 'paid' || o.status === 'active' || o.status === 'completed'
                            ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                            : 'bg-amber-500/10 text-amber-400 border-amber-500/20'
                        }`}
                      >
                        {o.status}
                      </span>
                    </td>
                    <td className="py-3 px-4 font-mono text-slate-400">
                      {o.paymentReference || '—'}
                    </td>
                    <td className="py-3 px-4 text-slate-400">
                      {new Date(o.createdAt).toLocaleDateString()}
                    </td>
                    <td className="py-3 px-4 text-right">
                      {o.status === 'pending_payment' ? (
                        <button
                          onClick={() => {
                            setSelectedOrder(o);
                            setIsVerifyOpen(true);
                            setVerifyMsg(null);
                          }}
                          className="px-2.5 py-1 rounded-lg bg-[#00D26A] text-black font-semibold hover:bg-[#00ba5e] text-xs transition"
                        >
                          Verify & Activate
                        </button>
                      ) : (
                        <span className="text-[#00D26A] font-medium">Provisioned</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Verify & Activate Modal */}
      {isVerifyOpen && selectedOrder && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <form
            onSubmit={handleVerifyOrderPayment}
            className="bg-[#121614] border border-white/[0.08] rounded-xl max-w-md w-full p-6 space-y-4"
          >
            <div className="flex items-center justify-between border-b border-white/[0.06] pb-3">
              <h3 className="font-semibold text-sm text-white flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-[#00D26A]" />
                Confirm Payment & Trigger Provisioning
              </h3>
              <button
                type="button"
                onClick={() => setIsVerifyOpen(false)}
                className="text-slate-400 hover:text-white text-xs font-bold"
              >
                ✕
              </button>
            </div>

            {verifyMsg && (
              <div
                className={`p-3 rounded-lg text-xs ${
                  verifyMsg.startsWith('Error')
                    ? 'bg-rose-500/10 border border-rose-500/20 text-rose-400'
                    : 'bg-[#00D26A]/10 border border-[#00D26A]/20 text-[#00D26A]'
                }`}
              >
                {verifyMsg}
              </div>
            )}

            <div className="text-xs text-slate-300 space-y-1 bg-black/40 p-3 rounded-lg border border-white/[0.04]">
              <div>Order ID: <span className="font-mono text-white">{selectedOrder.id}</span></div>
              <div>Expected Amount: <span className="font-bold text-white">{selectedOrder.amount} {selectedOrder.currency}</span></div>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Payment Provider</label>
              <select
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
              >
                <option value="cbe" className="bg-[#121614]">CBE</option>
                <option value="telebirr" className="bg-[#121614]">Telebirr</option>
                <option value="boa" className="bg-[#121614]">Bank of Abyssinia</option>
                <option value="dashen" className="bg-[#121614]">Dashen Bank</option>
                <option value="awash" className="bg-[#121614]">Awash Bank</option>
                <option value="mpesa" className="bg-[#121614]">M-Pesa</option>
                <option value="mock" className="bg-[#121614]">Mock Sandbox</option>
              </select>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">
                Bank Transaction ID / Slip Reference
              </label>
              <input
                type="text"
                required
                value={txId}
                onChange={(e) => setTxId(e.target.value)}
                placeholder="e.g. FT262036S1GH10104 or TEST_SUCCESS"
                className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A] font-mono"
              />
            </div>

            <div className="pt-3 border-t border-white/[0.06] flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setIsVerifyOpen(false)}
                className="px-4 py-2 rounded-lg text-xs font-medium text-slate-400 hover:text-white transition"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={verifying}
                className="px-4 py-2 rounded-lg text-xs font-semibold bg-[#00D26A] text-black hover:bg-[#00ba5e] disabled:opacity-50 transition"
              >
                {verifying ? 'Verifying...' : 'Verify & Activate'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Create Order Modal */}
      {isCreateOpen && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <form
            onSubmit={handleCreateOrder}
            className="bg-[#121614] border border-white/[0.08] rounded-xl max-w-md w-full p-6 space-y-4"
          >
            <div className="flex items-center justify-between border-b border-white/[0.06] pb-3">
              <h3 className="font-semibold text-sm text-white">Create New Service Order</h3>
              <button
                type="button"
                onClick={() => setIsCreateOpen(false)}
                className="text-slate-400 hover:text-white text-xs font-bold"
              >
                ✕
              </button>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Service Type</label>
              <select
                value={serviceType}
                onChange={(e) => setServiceType(e.target.value)}
                className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
              >
                <option value="payment-verifier" className="bg-[#121614]">Payment Verifier Subscription</option>
                <option value="vps" className="bg-[#121614]">Cloud VPS Instance</option>
                <option value="app-hosting" className="bg-[#121614]">App Hosting Container</option>
              </select>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Service Tier</label>
              <input
                type="text"
                value={tier}
                onChange={(e) => setTier(e.target.value)}
                placeholder="e.g. standard or pro"
                className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Amount (ETB)</label>
              <input
                type="number"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
              />
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
                className="px-4 py-2 rounded-lg text-xs font-semibold bg-[#00D26A] text-black hover:bg-[#00ba5e] transition"
              >
                Create Order
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
