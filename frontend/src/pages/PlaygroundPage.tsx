import React, { useState } from 'react';
import {
  Terminal,
  Play,
  Copy,
  Check,
  AlertCircle,
  CheckCircle2,
  Code2,
  Lock,
  Layers,
  Sparkles,
  RefreshCw,
  Clock,
  ExternalLink,
  ChevronRight,
  ShieldCheck,
  Zap,
} from 'lucide-react';
import type { VerificationResult } from '../types';
import { api } from '../api/client';

interface PresetScenario {
  name: string;
  provider: string;
  txId: string;
  expectedAmount?: string;
  description: string;
  tag: 'live' | 'sandbox';
}

const PRESETS: PresetScenario[] = [
  {
    name: 'Abyssinia (BOA)',
    provider: 'boa',
    txId: 'FT23062669JJ90172',
    expectedAmount: '200',
    description: 'Live test slip on Bank of Abyssinia cs portal',
    tag: 'live',
  },
  {
    name: 'Awash Bank',
    provider: 'awash',
    txId: '-E41AE0D86FFA-21XYYW',
    expectedAmount: '100',
    description: 'Awash Birr Pro online receipt portal reference',
    tag: 'live',
  },
  {
    name: 'Dashen Bank',
    provider: 'dashen',
    txId: 'REF-DASHEN-2024-9981',
    expectedAmount: '500',
    description: 'Dashen receipts API validation format',
    tag: 'live',
  },
  {
    name: 'Telebirr SuperApp',
    provider: 'telebirr',
    txId: 'https://telebirr.et/receipt?transid=TB20240918',
    expectedAmount: '250',
    description: 'Telebirr official receipt URL or slip code',
    tag: 'live',
  },
  {
    name: 'M-Pesa Safaricom',
    provider: 'mpesa',
    txId: 'SKL098XYZ',
    expectedAmount: '350',
    description: 'Safaricom M-Pesa Ethiopia B2C / C2B transaction',
    tag: 'live',
  },
  {
    name: 'Mock (Success)',
    provider: 'mock',
    txId: 'TEST_SUCCESS',
    expectedAmount: '500',
    description: 'Deterministic 500 ETB verified response',
    tag: 'sandbox',
  },
  {
    name: 'Mock (Already Claimed)',
    provider: 'mock',
    txId: 'TEST_CLAIMED',
    expectedAmount: '500',
    description: 'Simulates slip claimed by order_existing_88',
    tag: 'sandbox',
  },
  {
    name: 'Mock (Insufficient)',
    provider: 'mock',
    txId: 'TEST_INSUFFICIENT',
    expectedAmount: '1000',
    description: 'Receipt for 500 ETB while expecting 1,000 ETB',
    tag: 'sandbox',
  },
];

const PROVIDER_OPTIONS = [
  { value: 'cbe', label: 'Commercial Bank of Ethiopia (CBE)' },
  { value: 'telebirr', label: 'Telebirr (Ethio Telecom)' },
  { value: 'boa', label: 'Bank of Abyssinia (BOA)' },
  { value: 'dashen', label: 'Dashen Bank' },
  { value: 'awash', label: 'Awash Bank (Awash Birr)' },
  { value: 'mpesa', label: 'M-Pesa Safaricom Ethiopia' },
  { value: 'mock', label: 'Mock Verification Engine (Sandbox)' },
];

export const PlaygroundPage: React.FC = () => {
  const [provider, setProvider] = useState('boa');
  const [txId, setTxId] = useState('FT23062669JJ90172');
  const [expectedAmount, setExpectedAmount] = useState('200');
  const [autoClaim, setAutoClaim] = useState(false);
  const [claimedBy, setClaimedBy] = useState('order_demo_101');
  const [customApiKey, setCustomApiKey] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<VerificationResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [executionLatency, setExecutionLatency] = useState<number | null>(null);
  const [httpStatus, setHttpStatus] = useState<number | null>(null);
  const [copiedCode, setCopiedCode] = useState(false);
  const [activeTab, setActiveTab] = useState<'telemetry' | 'code'>('telemetry');
  const [codeLang, setCodeLang] = useState<'curl' | 'ts' | 'python' | 'go'>('curl');

  const handleSelectPreset = (preset: PresetScenario) => {
    setProvider(preset.provider);
    setTxId(preset.txId);
    if (preset.expectedAmount) {
      setExpectedAmount(preset.expectedAmount);
    }
  };

  const handleRunVerify = async () => {
    setLoading(true);
    setError(null);
    setResult(null);
    setHttpStatus(null);
    setExecutionLatency(null);

    const startTime = performance.now();
    try {
      const res = await api.verifyPayment({
        provider,
        transactionId: txId.trim(),
        expectedAmount: expectedAmount ? parseFloat(expectedAmount) : undefined,
        autoClaim,
        claimedBy: autoClaim ? claimedBy.trim() : undefined,
        apiKey: customApiKey.trim() || undefined,
      });
      const duration = Math.round(performance.now() - startTime);
      setExecutionLatency(duration);
      setHttpStatus(200);
      setResult(res);
      setActiveTab('telemetry');
    } catch (err: any) {
      const duration = Math.round(performance.now() - startTime);
      setExecutionLatency(duration);
      setHttpStatus(err.status || 400);
      setError(err.message || 'Verification request failed');
      setActiveTab('telemetry');
    } finally {
      setLoading(false);
    }
  };

  const generateCode = () => {
    const origin = window.location.origin;
    const endpoint = `${origin}/api/v1/public/payment/verify`;
    const payload: any = {
      provider,
      transactionId: txId,
      expectedAmount: expectedAmount ? parseFloat(expectedAmount) : undefined,
      autoClaim,
    };
    if (autoClaim) {
      payload.claimedBy = claimedBy;
    }

    if (codeLang === 'curl') {
      const headerStr = customApiKey ? ` \\\n  -H "X-API-KEY: ${customApiKey}"` : '';
      return `curl -X POST "${endpoint}" \\
  -H "Content-Type: application/json"${headerStr} \\
  -d '${JSON.stringify(payload, null, 2)}'`;
    }

    if (codeLang === 'ts') {
      return `import axios from 'axios';

const { data } = await axios.post('${endpoint}', ${JSON.stringify(payload, null, 2)}, {
  headers: {
    'Content-Type': 'application/json',
${customApiKey ? `    'X-API-KEY': '${customApiKey}',\n` : ''}  }
});

console.log('Verified:', data.verified);
console.log('Payment:', data.payment);`;
    }

    if (codeLang === 'python') {
      return `import requests

url = "${endpoint}"
headers = {
    "Content-Type": "application/json",
${customApiKey ? `    "X-API-KEY": "${customApiKey}",\n` : ''}}
payload = ${JSON.stringify(payload, null, 4)}

response = requests.post(url, json=payload, headers=headers)
data = response.json()
print("Verified:", data.get("verified"))
print("Settled Amount:", data.get("payment", {}).get("amount"))`;
    }

    // Go
    return `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	payload, _ := json.Marshal(map[string]interface{}{
		"provider":        "${provider}",
		"transactionId":   "${txId}",
		"expectedAmount":  ${expectedAmount || 0},
		"autoClaim":       ${autoClaim},
	})

	req, _ := http.NewRequest("POST", "${endpoint}", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
${customApiKey ? `	req.Header.Set("X-API-KEY", "${customApiKey}")\n` : ''}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Response:", string(body))
}`;
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCode(true);
    setTimeout(() => setCopiedCode(false), 2000);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-lg font-semibold text-white">Playground</h1>
      </div>

      {/* Quick Scenario Presets */}
      <div className="p-3.5 rounded-xl bg-[#121614] border border-white/[0.06]">
        <div className="flex items-center gap-2 mb-2 text-xs font-medium text-slate-400">
          <span>Presets:</span>
        </div>
        <div className="flex flex-wrap gap-2">
          {PRESETS.map((p) => (
            <button
              key={p.name}
              type="button"
              onClick={() => handleSelectPreset(p)}
              className="px-2.5 py-1 text-xs rounded-lg bg-white/[0.04] hover:bg-white/[0.08] border border-white/[0.06] text-slate-300 hover:text-white transition flex items-center gap-1.5"
            >
              <span
                className={`w-1.5 h-1.5 rounded-full ${
                  p.tag === 'live' ? 'bg-[#00D26A]' : 'bg-slate-400'
                }`}
              />
              <span className="font-medium">{p.name}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Main Split Layout: Request Builder on Left, Response & Code on Right */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Column: Request Form (5 Cols) */}
        <div className="lg:col-span-5 bg-[#121614] border border-white/[0.06] rounded-xl p-5 space-y-4">
          <div className="flex items-center justify-between border-b border-white/[0.06] pb-3">
            <h2 className="text-sm font-bold text-white flex items-center gap-2">
              <Zap className="w-4 h-4 text-[#00D26A]" />
              Request Configuration
            </h2>
            <span className="text-[10px] font-mono text-slate-500">POST /payment/verify</span>
          </div>

          {/* Provider Selector */}
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-300">Banking Rail / Provider</label>
            <select
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
            >
              {PROVIDER_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value} className="bg-[#121614] text-white">
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* Reference / Transaction ID Input */}
          <div className="space-y-1.5">
            <div className="flex justify-between items-center">
              <label className="text-xs font-semibold text-slate-300">
                Transaction Reference / Slip Code / URL
              </label>
            </div>
            <input
              type="text"
              value={txId}
              onChange={(e) => setTxId(e.target.value)}
              placeholder="e.g. FT23062669JJ90172 or https://telebirr.et/receipt?..."
              className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
            />
          </div>

          {/* Expected Amount */}
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-300">
              Expected Amount (ETB)
              <span className="text-[10px] font-normal text-slate-500 ml-1">Optional exact match</span>
            </label>
            <div className="relative">
              <input
                type="number"
                step="any"
                value={expectedAmount}
                onChange={(e) => setExpectedAmount(e.target.value)}
                placeholder="Leave blank to accept any amount"
                className="w-full pl-3 pr-12 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
              />
              <span className="absolute right-3 top-2 text-[10px] text-slate-500 font-mono font-bold">
                ETB
              </span>
            </div>
          </div>

          {/* Atomic Auto-Claim Options */}
          <div className="pt-2 border-t border-white/[0.06] space-y-3">
            <label className="flex items-center gap-2 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={autoClaim}
                onChange={(e) => setAutoClaim(e.target.checked)}
                className="rounded bg-black/40 border-white/[0.1] text-[#00D26A] focus:ring-0 focus:ring-offset-0"
              />
              <span className="text-xs font-semibold text-slate-200">
                Atomically Claim on Verification
              </span>
            </label>

            {autoClaim && (
              <div className="space-y-1.5 pl-6">
                <label className="text-[11px] font-medium text-slate-400">Claimant Identifier</label>
                <input
                  type="text"
                  value={claimedBy}
                  onChange={(e) => setClaimedBy(e.target.value)}
                  placeholder="e.g. order_8921"
                  className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
                />
              </div>
            )}
          </div>

          {/* API Key Header */}
          <div className="pt-2 border-t border-white/[0.06] space-y-1.5">
            <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
              <Lock className="w-3.5 h-3.5 text-slate-400" />
              API Key (X-API-KEY Header)
            </label>
            <input
              type="password"
              value={customApiKey}
              onChange={(e) => setCustomApiKey(e.target.value)}
              placeholder="Defaults to current session token"
              className="w-full px-3 py-1.5 rounded-lg bg-black/40 border border-white/[0.08] text-xs font-mono text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
            />
          </div>

          {/* Execute Button */}
          <button
            type="button"
            onClick={handleRunVerify}
            disabled={loading || !txId.trim()}
            className="w-full py-2.5 rounded-lg bg-[#00D26A] hover:bg-[#00ba5e] disabled:opacity-40 text-black text-xs font-semibold flex items-center justify-center gap-2 transition"
          >
            {loading ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin text-black" />
                Querying Bank Rail...
              </>
            ) : (
              <>
                <Play className="w-4 h-4 fill-black text-black" />
                Execute Verification Call
              </>
            )}
          </button>
        </div>

        {/* Right Column: Results & Code Generator (7 Cols) */}
        <div className="lg:col-span-7 bg-[#121614] border border-white/[0.06] rounded-xl overflow-hidden flex flex-col">
          {/* Tabs Bar */}
          <div className="flex items-center justify-between border-b border-white/[0.06] px-4 bg-black/20">
            <div className="flex items-center gap-1">
              <button
                type="button"
                onClick={() => setActiveTab('telemetry')}
                className={`py-3 px-3 text-xs font-semibold border-b-2 transition flex items-center gap-1.5 ${
                  activeTab === 'telemetry'
                    ? 'border-[#00D26A] text-white'
                    : 'border-transparent text-slate-400 hover:text-slate-200'
                }`}
              >
                <Terminal className="w-3.5 h-3.5" />
                Response Telemetry
              </button>
              <button
                type="button"
                onClick={() => setActiveTab('code')}
                className={`py-3 px-3 text-xs font-semibold border-b-2 transition flex items-center gap-1.5 ${
                  activeTab === 'code'
                    ? 'border-[#00D26A] text-white'
                    : 'border-transparent text-slate-400 hover:text-slate-200'
                }`}
              >
                <Code2 className="w-3.5 h-3.5" />
                Client SDKs & cURL
              </button>
            </div>

            {/* Execution Meta (Latency & HTTP Status) */}
            {httpStatus !== null && (
              <div className="flex items-center gap-2 text-xs">
                {executionLatency !== null && (
                  <span className="flex items-center gap-1 text-slate-400 font-mono text-[11px]">
                    <Clock className="w-3 h-3 text-slate-500" />
                    {executionLatency}ms
                  </span>
                )}
                <span
                  className={`px-2 py-0.5 rounded text-[10px] font-bold font-mono ${
                    httpStatus === 200
                      ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                      : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                  }`}
                >
                  HTTP {httpStatus}
                </span>
              </div>
            )}
          </div>

          {/* Tab 1: Response Telemetry */}
          {activeTab === 'telemetry' && (
            <div className="p-5 space-y-4 min-h-[420px] flex flex-col justify-between">
              {loading ? (
                <div className="my-auto text-center space-y-3 py-16">
                  <div className="inline-block h-8 w-8 rounded-full border-2 border-[#00D26A] border-t-transparent animate-spin" />
                  <div className="text-xs text-slate-400 font-medium">
                    Connecting to {provider.toUpperCase()} verification gateway...
                  </div>
                </div>
              ) : error ? (
                <div className="space-y-3">
                  <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs space-y-2">
                    <div className="flex items-center gap-2 font-bold text-rose-400">
                      <AlertCircle className="w-4 h-4" />
                      Verification Request Rejected
                    </div>
                    <div className="font-mono text-[11px]">{error}</div>
                  </div>
                </div>
              ) : result ? (
                <div className="space-y-4">
                  {/* Status Banner */}
                  <div
                    className={`p-3.5 rounded-xl border flex items-center justify-between ${
                      result.verified
                        ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-300'
                        : 'bg-amber-500/10 border-amber-500/20 text-amber-300'
                    }`}
                  >
                    <div className="flex items-center gap-2.5">
                      {result.verified ? (
                        <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />
                      ) : (
                        <AlertCircle className="w-5 h-5 text-amber-400 shrink-0" />
                      )}
                      <div>
                        <div className="font-bold text-xs">
                          {result.verified
                            ? 'Verification Succeeded'
                            : 'Slip Not Verified'}
                        </div>
                        <div className="text-[11px] opacity-80">
                          {result.errorMessage || result.warningMessage || (result.verified ? 'Receipt authenticity confirmed on banking rail.' : 'Invalid receipt or mismatch.')}
                        </div>
                      </div>
                    </div>
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-black/40 font-bold uppercase">
                      {result.verified ? 'VALID' : 'INVALID'}
                    </span>
                  </div>

                  {/* Settled Slip Details */}
                  {result.verified && (
                    <div className="p-4 rounded-xl bg-black/40 border border-white/[0.04] space-y-2.5 text-xs">
                      <div className="flex justify-between items-center text-slate-400">
                        <span>Settled Amount:</span>
                        <span className="font-bold text-white text-sm">
                          {result.amount.toLocaleString()} {result.currency || 'ETB'}
                        </span>
                      </div>
                      <div className="flex justify-between items-center text-slate-400">
                        <span>Payer:</span>
                        <span className="text-slate-200 font-medium">
                          {result.payerIdentifier || '—'}
                        </span>
                      </div>
                      <div className="flex justify-between items-center text-slate-400">
                        <span>Beneficiary:</span>
                        <span className="text-slate-200 font-medium">
                          {result.receiverIdentifier || '—'}
                        </span>
                      </div>
                      <div className="flex justify-between items-center text-slate-400">
                        <span>Claim State:</span>
                        <span className="font-medium text-slate-200">
                          {result.claimed ? (
                            <span className="text-[#00D26A]">Claimed ({result.claimedBy || 'system'})</span>
                          ) : (
                            <span className="text-amber-400">Unclaimed</span>
                          )}
                        </span>
                      </div>
                    </div>
                  )}

                  {/* Raw JSON viewer */}
                  <div className="space-y-1.5">
                    <div className="flex justify-between items-center">
                      <span className="text-[11px] font-semibold text-slate-400">
                        Complete JSON Payload
                      </span>
                      <button
                        type="button"
                        onClick={() => copyToClipboard(JSON.stringify(result, null, 2))}
                        className="text-[10px] text-slate-400 hover:text-white flex items-center gap-1"
                      >
                        {copiedCode ? <Check className="w-3 h-3 text-[#00D26A]" /> : <Copy className="w-3 h-3" />}
                        Copy
                      </button>
                    </div>
                    <pre className="p-3.5 rounded-xl bg-black/40 border border-white/[0.04] text-[11px] font-mono text-slate-300 overflow-x-auto max-h-56 scrollbar-thin">
                      {JSON.stringify(result, null, 2)}
                    </pre>
                  </div>
                </div>
              ) : (
                <div className="my-auto text-center space-y-2 py-16 text-slate-500">
                  <Terminal className="w-10 h-10 mx-auto opacity-30" />
                  <div className="text-xs font-semibold text-slate-400">Awaiting execution</div>
                  <p className="text-[11px] max-w-xs mx-auto">
                    Configure your parameters on the left and click "Execute Verification Call" or pick a quick preset.
                  </p>
                </div>
              )}
            </div>
          )}

          {/* Tab 2: Code Snippets */}
          {activeTab === 'code' && (
            <div className="p-5 space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  {(['curl', 'ts', 'python', 'go'] as const).map((lang) => (
                    <button
                      key={lang}
                      type="button"
                      onClick={() => setCodeLang(lang)}
                      className={`px-2.5 py-1 rounded-md text-xs font-semibold uppercase transition ${
                        codeLang === lang
                          ? 'bg-[#00D26A] text-black font-bold'
                          : 'bg-white/[0.04] text-slate-400 hover:text-white border border-white/[0.06]'
                      }`}
                    >
                      {lang === 'ts' ? 'TypeScript' : lang}
                    </button>
                  ))}
                </div>

                <button
                  type="button"
                  onClick={() => copyToClipboard(generateCode())}
                  className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-white/[0.04] hover:bg-white/[0.08] text-slate-300 hover:text-white text-xs font-semibold border border-white/[0.06] transition"
                >
                  {copiedCode ? (
                    <>
                      <Check className="w-3.5 h-3.5 text-[#00D26A]" />
                      Copied!
                    </>
                  ) : (
                    <>
                      <Copy className="w-3.5 h-3.5" />
                      Copy Code
                    </>
                  )}
                </button>
              </div>

              <pre className="p-4 rounded-xl bg-black/40 border border-white/[0.04] text-[11px] font-mono text-slate-300 overflow-x-auto min-h-[300px] scrollbar-thin">
                {generateCode()}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
