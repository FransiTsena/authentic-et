import React, { useState, useEffect } from 'react';
import { ShieldCheck, LogIn, UserPlus, AlertCircle, Sparkles, CheckCircle2 } from 'lucide-react';
import type { User } from '../types';
import { api } from '../api/client';

interface LoginPageProps {
  onSuccess: (user: User) => void;
}

export const LoginPage: React.FC<LoginPageProps> = ({ onSuccess }) => {
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
  const [allowRegistration, setAllowRegistration] = useState(false);
  const [isRegister, setIsRegister] = useState(false);
  
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    api.getSetupStatus()
      .then((status) => {
        if (!mounted) return;
        setNeedsSetup(status.needsSetup);
        setAllowRegistration(status.allowRegistration);
        if (status.needsSetup) {
          setIsRegister(true);
        }
      })
      .catch(() => {
        // Fallback default to standard login if endpoint fails
        if (!mounted) return;
        setNeedsSetup(false);
      });
    return () => {
      mounted = false;
    };
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    if (needsSetup || isRegister) {
      if (needsSetup && password !== confirmPassword) {
        setError('Passwords do not match');
        setLoading(false);
        return;
      }
    }

    try {
      if (needsSetup || isRegister) {
        const res = await api.register(email.trim(), password, name.trim());
        onSuccess(res.user);
      } else {
        const res = await api.login(email.trim(), password);
        onSuccess(res.user);
      }
    } catch (err: any) {
      setError(err.message || 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  const isSetupMode = needsSetup === true;

  return (
    <div className="max-w-md mx-auto my-12 p-8 rounded-2xl bg-[#121614] border border-white/[0.08] space-y-6">
      <div className="text-center space-y-2">
        <div className="inline-flex h-12 w-12 rounded-2xl bg-[#00D26A]/10 border border-[#00D26A]/20 text-[#00D26A] items-center justify-center">
          {isSetupMode ? <Sparkles className="w-6 h-6" /> : <ShieldCheck className="w-6 h-6" />}
        </div>

        {isSetupMode ? (
          <div>
            <div className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[10px] font-semibold tracking-wide uppercase bg-[#00D26A]/10 border border-[#00D26A]/20 text-[#00D26A] mb-2">
              <CheckCircle2 className="w-3 h-3" /> Initial Deployment Setup
            </div>
            <h2 className="text-xl font-bold text-white tracking-tight">Create Root Administrator</h2>
            <p className="text-xs text-slate-400 mt-1">
              Welcome to authentic-et. No administrator account exists yet. Create your administrator credentials to initialize and secure this instance.
            </p>
          </div>
        ) : (
          <div>
            <h2 className="text-xl font-bold text-white tracking-tight">
              {isRegister ? 'Create authentic-et Account' : 'Sign in to authentic-et'}
            </h2>
            <p className="text-xs text-slate-400 mt-1">
              {isRegister
                ? 'Get your developer API key and manage payment verification'
                : 'Access administrative controls, verification logs, and outbox telemetry'}
            </p>
          </div>
        )}
      </div>

      {error && (
        <div className="p-3.5 rounded-lg bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-center gap-2">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        {(isSetupMode || isRegister) && (
          <div>
            <label className="block text-xs font-medium text-slate-400 mb-1">
              {isSetupMode ? 'Administrator Name' : 'Your Full Name'}
            </label>
            <input
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={isSetupMode ? 'e.g. Lead Administrator' : 'e.g. Alex Johnson'}
              className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
            />
          </div>
        )}

        <div>
          <label className="block text-xs font-medium text-slate-400 mb-1">
            {isSetupMode ? 'Root Administrator Email' : 'Email Address'}
          </label>
          <input
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={isSetupMode ? 'admin@yourcompany.com' : 'user@company.com'}
            className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
          />
        </div>

        <div>
          <label className="block text-xs font-medium text-slate-400 mb-1">
            {isSetupMode ? 'Master Password (min 8 chars)' : 'Password'}
          </label>
          <input
            type="password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••••••"
            className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
          />
        </div>

        {isSetupMode && (
          <div>
            <label className="block text-xs font-medium text-slate-400 mb-1">Confirm Master Password</label>
            <input
              type="password"
              required
              minLength={8}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="••••••••••••"
              className="w-full px-3 py-2 rounded-lg bg-black/40 border border-white/[0.08] text-xs text-white focus:outline-none focus:border-[#00D26A]"
            />
          </div>
        )}

        <button
          type="submit"
          disabled={loading}
          className="w-full py-2.5 rounded-lg text-black font-semibold text-xs bg-[#00D26A] hover:bg-[#00ba5e] transition disabled:opacity-50 flex items-center justify-center gap-2"
        >
          {loading ? (
            'Please wait...'
          ) : isSetupMode ? (
            <>
              <Sparkles className="w-4 h-4" /> Initialize authentic-et & Launch
            </>
          ) : isRegister ? (
            <>
              <UserPlus className="w-4 h-4" /> Create Account
            </>
          ) : (
            <>
              <LogIn className="w-4 h-4" /> Sign In
            </>
          )}
        </button>
      </form>

      {!isSetupMode && (
        <div className="text-center pt-2 border-t border-white/[0.06]">
          {allowRegistration ? (
            <button
              type="button"
              onClick={() => {
                setIsRegister(!isRegister);
                setError(null);
              }}
              className="text-xs text-[#00D26A] hover:underline font-medium"
            >
              {isRegister ? 'Already have an account? Sign In' : "Don't have an account? Register"}
            </button>
          ) : (
            <p className="text-[11px] text-slate-500">
              Self-registration is disabled. Request account access from your administrator.
            </p>
          )}
        </div>
      )}
    </div>
  );
};
