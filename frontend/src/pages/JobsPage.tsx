import React, { useEffect, useState } from 'react';
import {
  Server,
  RefreshCw,
  AlertTriangle,
  CheckCircle2,
  Clock,
  RotateCcw,
} from 'lucide-react';
import type { ProvisioningJob } from '../types';
import { api } from '../api/client';

export const JobsPage: React.FC = () => {
  const [jobs, setJobs] = useState<ProvisioningJob[]>([]);
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [retryingId, setRetryingId] = useState<string | null>(null);

  const fetchJobs = async () => {
    setLoading(true);
    try {
      const res = await api.listProvisioningJobs(statusFilter || undefined);
      setJobs(res.items || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchJobs();
  }, [statusFilter]);

  const handleRetry = async (jobId: string) => {
    setRetryingId(jobId);
    try {
      await api.retryProvisioningJob(jobId);
      fetchJobs();
    } catch (err: any) {
      alert(`Retry failed: ${err.message}`);
    } finally {
      setRetryingId(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-lg font-semibold text-white">Outbox</h1>
        </div>
        <div className="flex items-center gap-3">
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-slate-300 focus:outline-none focus:border-[#00D26A]"
          >
            <option value="" className="bg-[#121614]">All Job Statuses</option>
            <option value="pending" className="bg-[#121614]">Pending</option>
            <option value="running" className="bg-[#121614]">Running</option>
            <option value="completed" className="bg-[#121614]">Completed</option>
            <option value="failed" className="bg-[#121614]">Failed (Retrying)</option>
            <option value="manual_review" className="bg-[#121614]">Manual Review Required</option>
          </select>
          <button
            onClick={() => fetchJobs()}
            className="p-2 rounded-lg bg-white/[0.04] text-slate-400 hover:text-white border border-white/[0.06] hover:border-white/[0.12] transition"
            title="Refresh"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <div className="rounded-xl bg-[#121614] border border-white/[0.06] overflow-hidden">
        {loading ? (
          <div className="py-20 text-center text-xs text-slate-500">Loading provisioning jobs...</div>
        ) : jobs.length === 0 ? (
          <div className="py-20 text-center text-xs text-slate-500">
            No provisioning jobs found. Once orders are verified, activation jobs appear here!
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-white/[0.06] text-slate-400 font-medium bg-black/20">
                  <th className="py-3 px-4">Job Key (Idempotency)</th>
                  <th className="py-3 px-4">Service</th>
                  <th className="py-3 px-4">Action</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4">Attempts</th>
                  <th className="py-3 px-4">Last Error / Message</th>
                  <th className="py-3 px-4">Updated At</th>
                  <th className="py-3 px-4 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04]">
                {jobs.map((job) => (
                  <tr key={job.id} className="hover:bg-white/[0.02] transition">
                    <td className="py-3 px-4 font-mono font-medium text-slate-200">{job.key}</td>
                    <td className="py-3 px-4 uppercase text-[#00D26A] font-semibold">{job.serviceType}</td>
                    <td className="py-3 px-4 font-mono text-slate-300">{job.action}</td>
                    <td className="py-3 px-4">
                      <span
                        className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold border ${
                          job.status === 'completed'
                            ? 'bg-[#00D26A]/10 text-[#00D26A] border-[#00D26A]/20'
                            : job.status === 'running'
                            ? 'bg-blue-500/10 text-blue-400 border-blue-500/20 animate-pulse'
                            : job.status === 'failed'
                            ? 'bg-amber-500/10 text-amber-400 border-amber-500/20'
                            : job.status === 'manual_review'
                            ? 'bg-rose-500/10 text-rose-400 border-rose-500/20 font-bold'
                            : 'bg-white/[0.04] text-slate-400 border-white/[0.06]'
                        }`}
                      >
                        {job.status}
                      </span>
                    </td>
                    <td className="py-3 px-4 font-medium text-slate-300">
                      {job.attempts} / {job.maxAttempts}
                    </td>
                    <td className="py-3 px-4 text-slate-400 max-w-xs truncate" title={job.lastError}>
                      {job.lastError || '—'}
                    </td>
                    <td className="py-3 px-4 text-slate-400">
                      {new Date(job.updatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </td>
                    <td className="py-3 px-4 text-right">
                      {(job.status === 'failed' || job.status === 'manual_review') && (
                        <button
                          onClick={() => handleRetry(job.id)}
                          disabled={retryingId === job.id}
                          className="flex items-center gap-1.5 ml-auto px-2.5 py-1 rounded-lg bg-white/[0.04] text-amber-400 hover:text-white hover:bg-white/[0.08] border border-white/[0.06] text-xs transition"
                          title="Retry Now"
                        >
                          <RotateCcw className={`w-3 h-3 ${retryingId === job.id ? 'animate-spin' : ''}`} />
                          Retry
                        </button>
                      )}
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
