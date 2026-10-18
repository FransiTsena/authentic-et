import React, { useEffect, useState } from 'react';
import { History, RefreshCw, ShieldCheck } from 'lucide-react';
import type { AuditEvent } from '../types';
import { api } from '../api/client';

export const AuditLogsPage: React.FC = () => {
  const [logs, setLogs] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const res = await api.listAuditLogs({ limit: 50 });
      setLogs(res.items || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold text-white">Audit Logs</h1>
        </div>
        <button
          onClick={fetchLogs}
          className="p-2 rounded-lg bg-white/[0.04] text-slate-400 hover:text-white border border-white/[0.06] hover:border-white/[0.12] transition"
          title="Refresh"
        >
          <RefreshCw className="w-3.5 h-3.5" />
        </button>
      </div>

      <div className="rounded-xl bg-[#121614] border border-white/[0.06] overflow-hidden">
        {loading ? (
          <div className="py-20 text-center text-xs text-slate-500">Loading audit trail...</div>
        ) : logs.length === 0 ? (
          <div className="py-20 text-center text-xs text-slate-500">No audit events recorded yet.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-white/[0.06] text-slate-400 font-medium bg-black/20">
                  <th className="py-3 px-4">Action</th>
                  <th className="py-3 px-4">Actor</th>
                  <th className="py-3 px-4">Resource</th>
                  <th className="py-3 px-4">Details</th>
                  <th className="py-3 px-4">Timestamp</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04]">
                {logs.map((log) => (
                  <tr key={log.id} className="hover:bg-white/[0.02] transition">
                    <td className="py-3 px-4 font-mono font-semibold text-[#00D26A]">{log.action}</td>
                    <td className="py-3 px-4 text-slate-300">
                      <span className="capitalize">{log.actorType}</span>
                      {log.actorId && <span className="text-[10px] text-slate-500 ml-1">({log.actorId})</span>}
                    </td>
                    <td className="py-3 px-4 text-slate-300">
                      <span className="capitalize text-slate-400">{log.resourceType}:</span>{' '}
                      <span className="font-mono text-slate-200">{log.resourceId}</span>
                    </td>
                    <td className="py-3 px-4 font-mono text-[11px] text-slate-400 max-w-xs truncate">
                      {log.details ? JSON.stringify(log.details) : '—'}
                    </td>
                    <td className="py-3 px-4 text-slate-400">
                      {new Date(log.createdAt).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};
