import React, { useState, useEffect } from 'react';
import {
  ShieldCheck,
  Zap,
  CheckCircle2,
  ArrowRight,
  Terminal,
  ExternalLink,
  Lock,
  Layers,
  Sparkles,
  Server,
  Database,
  Clock,
  AlertTriangle,
  Play,
  RefreshCw,
} from 'lucide-react';
import type { User, VerificationResult } from '../types';
import { api } from '../api/client';
import { BrandLogo } from '../components/BrandLogo';

interface PublicLandingPageProps {
  onLoginSuccess: (user: User) => void;
  onOpenDocs?: () => void;
  onOpenHealth?: () => void;
}

const BANK_RAILS = [
  {
    key: 'cbe',
    name: 'Commercial Bank of Ethiopia',
    short: 'CBE Birr & Mobile',
    tag: 'National Rail',
    accent: 'text-purple-400',
    desc: 'Validates v2 receipt slip URLs and automated compound merchant name matching.',
  },
  {
    key: 'telebirr',
    name: 'Telebirr SuperApp',
    short: 'Ethio Telecom',
    tag: 'Mobile Wallet',
    accent: 'text-sky-400',
    desc: 'Direct receipt validation with bilingual Amharic/English slip parsing.',
  },
  {
    key: 'boa',
    name: 'Bank of Abyssinia',
    short: 'BOA Online Slip',
    tag: 'Commercial Bank',
    accent: 'text-amber-400',
    desc: 'Live online slip validation with payer name, beneficiary, and fee settlement.',
  },
  {
    key: 'dashen',
    name: 'Dashen Bank',
    short: 'Dashen SuperApp',
    tag: 'Commercial Bank',
    accent: 'text-blue-400',
    desc: 'JSON receipts API validation directly with Dashen banking gateway.',
  },
  {
    key: 'awash',
    name: 'Awash Bank',
    short: 'Awash Birr Pro',
    tag: 'Commercial Bank',
    accent: 'text-orange-400',
    desc: 'Online receipt portal scraping and verification on secure banking ports.',
  },
  {
    key: 'mpesa',
    name: 'M-Pesa Safaricom',
    short: 'Safaricom Ethiopia',
    tag: 'Telco Money',
    accent: 'text-emerald-400',
    desc: 'B2C and C2B transaction slips with pure Go zlib stream decompression.',
  },
];

export const PublicLandingPage: React.FC<PublicLandingPageProps> = ({
  onLoginSuccess,
  onOpenDocs,
  onOpenHealth,
}) => {
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
  const [showAuthModal, setShowAuthModal] = useState(false);
  const [authMode, setAuthMode] = useState<'login' | 'register'>('login');

  // Auth form states
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [name, setName] = useState('');
  const [authLoading, setAuthLoading] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);

  // Live Verifier Interactive Demo states
  const [demoProvider, setDemoProvider] = useState('boa');
  const [demoTxId, setDemoTxId] = useState('FT23062669JJ90172');
  const [demoExpectedAmount, setDemoExpectedAmount] = useState('200');
  const [demoLoading, setDemoLoading] = useState(false);
  const [demoResult, setDemoResult] = useState<VerificationResult | null>(null);
  const [demoError, setDemoError] = useState<string | null>(null);
  const [demoLatency, setDemoLatency] = useState<number | null>(null);

  useEffect(() => {
    api.getSetupStatus()
      .then((status) => {
        setNeedsSetup(status.needsSetup);
        if (status.needsSetup) {
          setAuthMode('register');
        }
      })
      .catch(() => setNeedsSetup(false));
  }, []);

  const handleAuthSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setAuthLoading(true);
    setAuthError(null);

    if (needsSetup || authMode === 'register') {
      if (needsSetup && password !== confirmPassword) {
        setAuthError('Passwords do not match');
        setAuthLoading(false);
        return;
      }
    }

    try {
      if (needsSetup || authMode === 'register') {
        const res = await api.register(email.trim(), password, name.trim());
        onLoginSuccess(res.user);
      } else {
        const res = await api.login(email.trim(), password);
        onLoginSuccess(res.user);
      }
    } catch (err: any) {
      setAuthError(err.message || 'Authentication failed');
    } finally {
      setAuthLoading(false);
    }
  };

  const handleRunDemoVerify = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    setDemoLoading(true);
    setDemoError(null);
    setDemoResult(null);

    const startTime = performance.now();
    try {
      const res = await api.verifyPayment({
        provider: demoProvider,
        transactionId: demoTxId.trim(),
        expectedAmount: demoExpectedAmount ? parseFloat(demoExpectedAmount) : undefined,
      });
      setDemoLatency(Math.round(performance.now() - startTime));
      setDemoResult(res);
    } catch (err: any) {
      setDemoLatency(Math.round(performance.now() - startTime));
      setDemoError(err.message || 'Verification request failed');
    } finally {
      setDemoLoading(false);
    }
  };

  const loadPreset = (provider: string, txId: string, amount: string) => {
    setDemoProvider(provider);
    setDemoTxId(txId);
    setDemoExpectedAmount(amount);
  };

  return (
    <div className="min-h-screen bg-[#0c0f0d] text-slate-100 flex flex-col font-sans selection:bg-[#00D26A] selection:text-black">
      {/* 1. Header (Clean Flat Navigation) */}
      <header className="sticky top-0 z-50 border-b border-white/[0.06] bg-[#0c0f0d]/90 backdrop-blur-md">
        <div className="max-w-[1200px] mx-auto px-5 sm:px-8 h-16 flex items-center justify-between">
          {/* Brand Logo */}
          <a href="#" className="group">
            <BrandLogo size={36} showText={true} />
          </a>

          {/* Center Links (Desktop) */}
          <nav className="hidden md:flex items-center gap-1 text-xs font-medium text-slate-400">
            <a
              href="#rails"
              className="px-3.5 py-1.5 rounded-lg hover:text-white hover:bg-white/[0.04] transition"
            >
              Banking Rails
            </a>
            <a
              href="#interactive-demo"
              className="px-3.5 py-1.5 rounded-lg hover:text-white hover:bg-white/[0.04] transition"
            >
              Live Verifier
            </a>
            {onOpenDocs && (
              <button
                onClick={onOpenDocs}
                className="px-3.5 py-1.5 rounded-lg hover:text-white hover:bg-white/[0.04] transition"
              >
                Documentation
              </button>
            )}
            {onOpenHealth && (
              <button
                onClick={onOpenHealth}
                className="px-3.5 py-1.5 rounded-lg hover:text-white hover:bg-white/[0.04] transition flex items-center gap-1.5"
              >
                <span className="w-1.5 h-1.5 rounded-full bg-[#00D26A]" />
                <span>Health</span>
              </button>
            )}
          </nav>

          {/* Right Action: Access Console Button */}
          <div className="flex items-center gap-3">
            <button
              onClick={() => {
                setShowAuthModal(true);
                setAuthError(null);
              }}
              className={`inline-flex items-center gap-2 rounded-lg font-semibold transition active:scale-95 h-9 px-4 text-xs ${
                needsSetup
                  ? 'bg-[#00D26A] text-black hover:bg-[#25F28A]'
                  : 'bg-white text-[#0c0f0d] hover:bg-slate-200'
              }`}
            >
              {needsSetup ? (
                <>
                  <Sparkles className="w-3.5 h-3.5" />
                  Initialize Root Admin
                </>
              ) : (
                <>
                  <Lock className="w-3.5 h-3.5" />
                  Operator Sign In
                </>
              )}
            </button>
          </div>
        </div>
      </header>

      {/* 2. Hero Section */}
      <section className="pt-16 pb-16 md:pt-24 md:pb-24 relative">
        <div className="max-w-[1200px] mx-auto px-5 sm:px-8 relative z-10">
          <div className="max-w-2xl mx-auto text-center space-y-5">
            {/* Pill Eyebrow */}
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium bg-white/[0.04] text-slate-300">
              <span className="w-2 h-2 rounded-full bg-[#00D26A]" />
              <span>Multi-Rail Payment Verification Platform for Ethiopia</span>
            </div>

            {/* Main Headline */}
            <h1 className="text-4xl sm:text-5xl md:text-6xl font-bold tracking-tight text-white leading-[1.08]">
              Every payment slip in Ethiopia,{' '}
              <span className="text-[#00D26A]">verified.</span>
            </h1>

            {/* Subtitle */}
            <p className="text-sm sm:text-base text-slate-400 leading-relaxed max-w-xl mx-auto">
              Connect Commercial Bank of Ethiopia (CBE), Telebirr, Abyssinia, Dashen, Awash Birr, and M-Pesa to your apps with guaranteed zero double-spending, pure Go speed, and transactional outbox service provisioning.
            </p>

            {/* Action Buttons */}
            <div className="pt-2 flex flex-wrap items-center justify-center gap-2.5">
              <a
                href="#interactive-demo"
                className="inline-flex items-center justify-center gap-2 rounded-lg font-semibold transition active:scale-95 h-10 px-5 text-xs bg-[#00D26A] text-black hover:bg-[#25F28A]"
              >
                <Zap className="w-3.5 h-3.5 fill-black" />
                Try Interactive Verifier
              </a>
              <button
                type="button"
                onClick={() => {
                  setShowAuthModal(true);
                  setAuthError(null);
                }}
                className="inline-flex items-center justify-center gap-2 rounded-lg font-semibold transition active:scale-95 h-10 px-5 text-xs text-slate-200 bg-white/[0.05] hover:bg-white/[0.09]"
              >
                {needsSetup ? 'Setup Administrator Account' : 'Sign in to Operator Console'}
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>

          {/* 3. Interactive Verifier macOS Window Demo (Clean Flat Window) */}
          <div id="interactive-demo" className="mt-12 max-w-3xl mx-auto scroll-mt-24">
            <div className="rounded-xl border border-white/[0.07] bg-[#121614] overflow-hidden">
              {/* Window Titlebar */}
              <div className="flex h-10 items-center justify-between border-b border-white/[0.06] bg-[#0d100e] px-4">
                <div className="flex items-center gap-1.5">
                  <span className="h-2.5 w-2.5 rounded-full bg-slate-700" />
                  <span className="h-2.5 w-2.5 rounded-full bg-slate-700" />
                  <span className="h-2.5 w-2.5 rounded-full bg-slate-700" />
                </div>
                <div className="flex items-center gap-2 text-xs font-mono text-slate-400 bg-black/40 px-3 py-1 rounded">
                  <Lock className="w-3 h-3 text-[#00D26A]" />
                  <span>https://authentic-et.local/api/v1/public/payment/verify</span>
                </div>
                <div className="w-12 text-right">
                  <span className="text-[10px] font-mono text-[#00D26A] bg-[#00D26A]/10 px-2 py-0.5 rounded">
                    ONLINE
                  </span>
                </div>
              </div>

              {/* Window Body: Interactive Verifier */}
              <div className="p-5 sm:p-6 space-y-5">
                <div>
                  <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider mb-2">
                    Quick Scenario Presets:
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    <button
                      type="button"
                      onClick={() => loadPreset('boa', 'FT23062669JJ90172', '200')}
                      className="px-2.5 py-1 rounded text-xs bg-white/[0.04] hover:bg-white/[0.08] text-slate-300 transition flex items-center gap-1.5"
                    >
                      <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />
                      Abyssinia (BOA) Live Slip
                    </button>
                    <button
                      type="button"
                      onClick={() => loadPreset('awash', '-E41AE0D86FFA-21XYYW', '100')}
                      className="px-2.5 py-1 rounded text-xs bg-white/[0.04] hover:bg-white/[0.08] text-slate-300 transition flex items-center gap-1.5"
                    >
                      <span className="w-1.5 h-1.5 rounded-full bg-orange-400" />
                      Awash Birr Pro
                    </button>
                    <button
                      type="button"
                      onClick={() => loadPreset('dashen', 'REF-DASHEN-2024-9981', '500')}
                      className="px-2.5 py-1 rounded text-xs bg-white/[0.04] hover:bg-white/[0.08] text-slate-300 transition flex items-center gap-1.5"
                    >
                      <span className="w-1.5 h-1.5 rounded-full bg-blue-400" />
                      Dashen Bank
                    </button>
                    <button
                      type="button"
                      onClick={() => loadPreset('telebirr', 'https://telebirr.et/receipt?transid=TB20240918', '250')}
                      className="px-2.5 py-1 rounded text-xs bg-white/[0.04] hover:bg-white/[0.08] text-slate-300 transition flex items-center gap-1.5"
                    >
                      <span className="w-1.5 h-1.5 rounded-full bg-sky-400" />
                      Telebirr Receipt
                    </button>
                    <button
                      type="button"
                      onClick={() => loadPreset('mock', 'TEST_SUCCESS', '500')}
                      className="px-2.5 py-1 rounded text-xs bg-white/[0.04] hover:bg-white/[0.08] text-slate-300 transition flex items-center gap-1.5"
                    >
                      <span className="w-1.5 h-1.5 rounded-full bg-[#00D26A]" />
                      Mock Sandbox (500 ETB)
                    </button>
                  </div>
                </div>

                {/* Form Input Grid */}
                <form onSubmit={handleRunDemoVerify} className="grid grid-cols-1 md:grid-cols-12 gap-2.5 items-end">
                  <div className="md:col-span-3 space-y-1">
                    <label className="text-xs font-medium text-slate-400">Rail</label>
                    <select
                      value={demoProvider}
                      onChange={(e) => setDemoProvider(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-black/50 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
                    >
                      <option value="boa">Abyssinia (BOA)</option>
                      <option value="cbe">Commercial Bank (CBE)</option>
                      <option value="telebirr">Telebirr SuperApp</option>
                      <option value="dashen">Dashen Bank</option>
                      <option value="awash">Awash Bank</option>
                      <option value="mpesa">M-Pesa Safaricom</option>
                      <option value="mock">Mock Sandbox Engine</option>
                    </select>
                  </div>

                  <div className="md:col-span-5 space-y-1">
                    <label className="text-xs font-medium text-slate-400">
                      Slip Reference / Transaction ID
                    </label>
                    <input
                      type="text"
                      value={demoTxId}
                      onChange={(e) => setDemoTxId(e.target.value)}
                      placeholder="e.g. FT23062669JJ90172"
                      className="w-full px-3 py-2 rounded-lg bg-black/50 border border-white/[0.08] text-xs font-mono text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
                    />
                  </div>

                  <div className="md:col-span-2 space-y-1">
                    <label className="text-xs font-medium text-slate-400">Expected</label>
                    <input
                      type="number"
                      step="any"
                      value={demoExpectedAmount}
                      onChange={(e) => setDemoExpectedAmount(e.target.value)}
                      placeholder="500"
                      className="w-full px-3 py-2 rounded-lg bg-black/50 border border-white/[0.08] text-xs font-mono text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
                    />
                  </div>

                  <div className="md:col-span-2">
                    <button
                      type="submit"
                      disabled={demoLoading || !demoTxId.trim()}
                      className="w-full py-2 rounded-lg bg-[#00D26A] hover:bg-[#25F28A] disabled:opacity-40 text-black text-xs font-bold transition flex items-center justify-center gap-1.5"
                    >
                      {demoLoading ? (
                        <>
                          <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                          Checking...
                        </>
                      ) : (
                        <>
                          <Play className="w-3.5 h-3.5 fill-black" />
                          Verify
                        </>
                      )}
                    </button>
                  </div>
                </form>

                {/* Demo Result Display */}
                {demoResult && (
                  <div className="p-3.5 rounded-lg bg-black/40 border border-white/[0.06] space-y-2.5">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        {demoResult.verified ? (
                          <CheckCircle2 className="w-4 h-4 text-[#00D26A]" />
                        ) : (
                          <AlertTriangle className="w-4 h-4 text-amber-400" />
                        )}
                        <span className="text-xs font-semibold text-white">
                          {demoResult.verified ? 'Official Slip Confirmed Valid' : 'Slip Not Found or Mismatch'}
                        </span>
                      </div>
                      {demoLatency !== null && (
                        <span className="text-[11px] font-mono text-slate-400 flex items-center gap-1">
                          <Clock className="w-3 h-3 text-slate-500" />
                          {demoLatency}ms
                        </span>
                      )}
                    </div>

                    {demoResult.verified && (
                      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1 text-xs">
                        <div className="bg-white/[0.02] p-2 rounded">
                          <div className="text-[10px] text-slate-400">Settled Amount</div>
                          <div className="font-bold text-[#00D26A] text-sm font-mono">
                            {demoResult.amount.toLocaleString()} {demoResult.currency || 'ETB'}
                          </div>
                        </div>
                        <div className="bg-white/[0.02] p-2 rounded">
                          <div className="text-[10px] text-slate-400">Payer Name / Number</div>
                          <div className="font-medium text-slate-200 truncate">
                            {demoResult.payerIdentifier || 'Official Sender'}
                          </div>
                        </div>
                        <div className="bg-white/[0.02] p-2 rounded">
                          <div className="text-[10px] text-slate-400">Beneficiary</div>
                          <div className="font-medium text-slate-200 truncate">
                            {demoResult.receiverIdentifier || 'Merchant Account'}
                          </div>
                        </div>
                        <div className="bg-white/[0.02] p-2 rounded">
                          <div className="text-[10px] text-slate-400">Claim State</div>
                          <div className="font-medium text-amber-400">
                            {demoResult.claimed ? 'Claimed' : 'Available'}
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {demoError && (
                  <div className="p-3 rounded-lg bg-rose-500/10 text-rose-300 text-xs flex items-center gap-2">
                    <AlertTriangle className="w-4 h-4 text-rose-400 shrink-0" />
                    <span>{demoError}</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* 4. Tabular Metrics Stats Row (Clean Flat Divider) */}
      <section className="py-12 border-y border-white/[0.06] bg-[#0f1210]">
        <div className="max-w-[1200px] mx-auto px-5 sm:px-8">
          <ul className="grid grid-cols-2 md:grid-cols-4 gap-6 text-center md:text-left">
            <li className="space-y-0.5">
              <p className="text-3xl sm:text-4xl font-bold text-[#00D26A] tracking-tight tabular-nums">
                6 Rails
              </p>
              <p className="text-sm font-semibold text-white">Live Banking Gateways</p>
              <p className="text-xs text-slate-400">
                CBE, Telebirr, Abyssinia, Dashen, Awash, M-Pesa.
              </p>
            </li>
            <li className="space-y-0.5">
              <p className="text-3xl sm:text-4xl font-bold text-white tracking-tight tabular-nums">
                ~15 MB
              </p>
              <p className="text-sm font-semibold text-white">Low Memory Footprint</p>
              <p className="text-xs text-slate-400">
                Runs effortlessly in micro Kubernetes app pods.
              </p>
            </li>
            <li className="space-y-0.5">
              <p className="text-3xl sm:text-4xl font-bold text-[#00D26A] tracking-tight tabular-nums">
                0
              </p>
              <p className="text-sm font-semibold text-white">Lost Cloud Orders</p>
              <p className="text-xs text-slate-400">
                Durable outbox with lease locks & retry backoff.
              </p>
            </li>
            <li className="space-y-0.5">
              <p className="text-3xl sm:text-4xl font-bold text-white tracking-tight tabular-nums">
                100%
              </p>
              <p className="text-sm font-semibold text-white">ACID State Machine</p>
              <p className="text-xs text-slate-400">
                Database-level double-spending prevention.
              </p>
            </li>
          </ul>
        </div>
      </section>

      {/* 5. Three Core Product Pillars (Clean Flat Cards) */}
      <section id="pillars" className="py-16 md:py-20">
        <div className="max-w-[1200px] mx-auto px-5 sm:px-8 space-y-10">
          <div className="max-w-xl">
            <p className="text-xs font-semibold text-[#00D26A] uppercase tracking-wider">
              Engineered for Ethiopian Production
            </p>
            <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-white mt-1">
              Three pillars of bulletproof financial automation.
            </h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
            {/* Pillar 1 */}
            <div className="rounded-xl bg-[#121614] border border-white/[0.06] p-6 flex flex-col justify-between hover:bg-[#151a17] transition">
              <div className="space-y-2.5">
                <div className="text-xs font-mono font-bold text-[#00D26A]">01</div>
                <h3 className="text-base font-bold text-white">Multi-Rail Verification</h3>
                <p className="text-xs text-slate-400 leading-relaxed">
                  Direct headless scraping and official JSON receipt endpoints for Commercial Bank of Ethiopia, Telebirr, Bank of Abyssinia, Dashen, Awash Birr, and Safaricom M-Pesa.
                </p>
              </div>
              <div className="pt-6 text-xs font-medium text-[#00D26A] flex items-center gap-1.5">
                <span>Atomic Slip Matching</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </div>
            </div>

            {/* Pillar 2 */}
            <div className="rounded-xl bg-[#121614] border border-white/[0.06] p-6 flex flex-col justify-between hover:bg-[#151a17] transition">
              <div className="space-y-2.5">
                <div className="text-xs font-mono font-bold text-[#00D26A]">02</div>
                <h3 className="text-base font-bold text-white">Zero Double-Spending</h3>
                <p className="text-xs text-slate-400 leading-relaxed">
                  Strict conditional updates (`WHERE claimed = 0`) prevent simultaneous race conditions. Once a slip is claimed by an order, any secondary attempt is immediately rejected.
                </p>
              </div>
              <div className="pt-6 text-xs font-medium text-[#00D26A] flex items-center gap-1.5">
                <span>Cryptographic Uniqueness</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </div>
            </div>

            {/* Pillar 3 */}
            <div className="rounded-xl bg-[#121614] border border-white/[0.06] p-6 flex flex-col justify-between hover:bg-[#151a17] transition">
              <div className="space-y-2.5">
                <div className="text-xs font-mono font-bold text-[#00D26A]">03</div>
                <h3 className="text-base font-bold text-white">Durable Outbox Queue</h3>
                <p className="text-xs text-slate-400 leading-relaxed">
                  Transactional outbox pattern decouples payment verification from cloud provisioning (e.g. VPS or SaaS). Background workers retry with exponential backoff and HMAC-signed webhooks.
                </p>
              </div>
              <div className="pt-6 text-xs font-medium text-[#00D26A] flex items-center gap-1.5">
                <span>Guaranteed Delivery</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* 6. Supported Rails Grid (Clean Minimal Cards) */}
      <section id="rails" className="py-16 border-t border-white/[0.06] bg-[#0f1210]">
        <div className="max-w-[1200px] mx-auto px-5 sm:px-8 space-y-8">
          <div className="max-w-lg">
            <h2 className="text-2xl font-bold text-white tracking-tight">
              Supported Ethiopian Banking Rails
            </h2>
            <p className="text-xs text-slate-400 mt-1">
              Active verification adapters ready out of the box with zero third-party subscriptions.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3.5">
            {BANK_RAILS.map((rail) => (
              <div
                key={rail.key}
                className="p-4 rounded-xl bg-[#121614] hover:bg-[#151a17] transition space-y-2"
              >
                <div className="flex items-center justify-between">
                  <span className={`text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded bg-white/[0.04] ${rail.accent}`}>
                    {rail.tag}
                  </span>
                  <span className="flex items-center gap-1 text-[11px] font-mono text-[#00D26A]">
                    <span className="w-1.5 h-1.5 rounded-full bg-[#00D26A]" />
                    Online
                  </span>
                </div>
                <div>
                  <h4 className="text-sm font-semibold text-white">{rail.name}</h4>
                  <div className="text-[11px] text-slate-400 font-mono">{rail.short}</div>
                </div>
                <p className="text-xs text-slate-400 leading-relaxed">{rail.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* 7. Footer (Minimal) */}
      <footer className="mt-auto border-t border-white/[0.06] py-8 bg-[#0c0f0d]">
        <div className="max-w-[1200px] mx-auto px-5 sm:px-8 flex flex-col md:flex-row items-center justify-between gap-4 text-xs text-slate-500">
          <div className="flex items-center gap-2">
            <span className="text-slate-300 font-semibold">authentic-et</span>
            <span>· Multi-Rail Payment Verification Platform</span>
          </div>
          <div className="flex items-center gap-4">
            {onOpenDocs && (
              <button onClick={onOpenDocs} className="hover:text-slate-300 transition">
                Documentation
              </button>
            )}
            {onOpenHealth && (
              <button onClick={onOpenHealth} className="hover:text-slate-300 transition">
                System Health
              </button>
            )}
            <a
              href="/api/v1/openapi.json"
              target="_blank"
              rel="noreferrer"
              className="hover:text-slate-300 transition"
            >
              OpenAPI 3.0 Spec
            </a>
            <span className="text-slate-700">|</span>
            <span>Apache 2.0</span>
          </div>
        </div>
      </footer>

      {/* 8. Dedicated Auth Modal (Clean Flat Surface, No Shadows) */}
      {showAuthModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm">
          <div className="relative w-full max-w-md rounded-2xl border border-white/[0.08] bg-[#121614] p-6 sm:p-7 space-y-5">
            <button
              onClick={() => setShowAuthModal(false)}
              className="absolute top-4 right-4 text-slate-400 hover:text-white text-xs font-bold p-1 rounded hover:bg-white/[0.06] transition"
            >
              ✕
            </button>

            <div className="text-center space-y-1.5">
              <div className="flex justify-center mb-1">
                <BrandLogo size={40} />
              </div>

              {needsSetup ? (
                <div>
                  <div className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[10px] font-semibold uppercase bg-[#00D26A]/10 text-[#00D26A] mb-1">
                    <CheckCircle2 className="w-3 h-3" /> Initial Deployment Setup
                  </div>
                  <h3 className="text-lg font-bold text-white tracking-tight">
                    Create Root Administrator
                  </h3>
                  <p className="text-xs text-slate-400 mt-0.5">
                    No administrator account exists yet. Create your administrator credentials to initialize this instance.
                  </p>
                </div>
              ) : (
                <div>
                  <h3 className="text-lg font-bold text-white tracking-tight">
                    Sign in to authentic-et Console
                  </h3>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Access real-time verification ledgers, outbox jobs, and API keys.
                  </p>
                </div>
              )}
            </div>

            {authError && (
              <div className="p-3 rounded-lg bg-rose-500/10 text-rose-300 text-xs flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 text-rose-400 shrink-0" />
                <span>{authError}</span>
              </div>
            )}

            <form onSubmit={handleAuthSubmit} className="space-y-3.5">
              {needsSetup && (
                <div className="space-y-1">
                  <label className="text-xs font-medium text-slate-300">Administrator Name</label>
                  <input
                    type="text"
                    required
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g. Lead Administrator"
                    className="w-full px-3 py-2 rounded-lg bg-black/50 border border-white/[0.08] text-xs text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
                  />
                </div>
              )}

              <div className="space-y-1">
                <label className="text-xs font-medium text-slate-300">Email Address</label>
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@yourcompany.com"
                  className="w-full px-3 py-2 rounded-lg bg-black/50 border border-white/[0.08] text-xs text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
                />
              </div>

              <div className="space-y-1">
                <label className="text-xs font-medium text-slate-300">Password</label>
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••••••"
                  className="w-full px-3 py-2 rounded-lg bg-black/50 border border-white/[0.08] text-xs text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
                />
              </div>

              {needsSetup && (
                <div className="space-y-1">
                  <label className="text-xs font-medium text-slate-300">Confirm Password</label>
                  <input
                    type="password"
                    required
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    placeholder="••••••••••••"
                    className="w-full px-3 py-2 rounded-lg bg-black/50 border border-white/[0.08] text-xs text-white placeholder-slate-600 focus:outline-none focus:border-[#00D26A]"
                  />
                </div>
              )}

              <button
                type="submit"
                disabled={authLoading}
                className="w-full py-2.5 rounded-lg bg-[#00D26A] hover:bg-[#25F28A] disabled:opacity-40 text-black text-xs font-bold transition flex items-center justify-center gap-2 mt-2"
              >
                {authLoading ? (
                  <>
                    <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                    Authenticating...
                  </>
                ) : needsSetup ? (
                  <>
                    <CheckCircle2 className="w-4 h-4" />
                    Initialize Root Administrator
                  </>
                ) : (
                  <>
                    <Lock className="w-3.5 h-3.5" />
                    Sign In to Console
                  </>
                )}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
