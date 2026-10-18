import React from 'react';

interface BrandLogoProps {
  size?: number;
  className?: string;
  showText?: boolean;
  theme?: 'dark' | 'light';
}

/**
 * authentic-et Minimalist Brand Logo
 * Flat, ultra-clean vector logo inspired by Upick's aerodynamic swoop curves,
 * customized for payment verification with zero artificial shadows or borders.
 */
export const BrandLogo: React.FC<BrandLogoProps> = ({
  size = 36,
  className = '',
  showText = false,
  theme = 'dark',
}) => {
  return (
    <div className={`inline-flex items-center gap-2.5 ${className}`}>
      <svg
        width={size}
        height={size}
        viewBox="0 0 100 100"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="shrink-0 transition-transform duration-200 group-hover:scale-105"
      >
        <defs>
          <linearGradient id="pgSwoopGrad" x1="15%" y1="80%" x2="85%" y2="20%">
            <stop offset="0%" stopColor="#079447" />
            <stop offset="100%" stopColor="#00D26A" />
          </linearGradient>

          <linearGradient id="pgArchGrad" x1="15%" y1="15%" x2="80%" y2="80%">
            <stop offset="0%" stopColor={theme === 'dark' ? '#F8FAFC' : '#0c0f0d'} />
            <stop offset="100%" stopColor={theme === 'dark' ? '#94A3B8' : '#475569'} />
          </linearGradient>
        </defs>

        {/* 
          1. The Gateway / P-Arch:
          Minimalist curved spine forming the backbone and arch
        */}
        <path
          d="M 28 66
             C 28 42, 30 26, 44 22
             C 54 19, 66 22, 68 32
             C 70 41, 62 48, 50 49
             C 41 50, 36 46, 36 46"
          stroke="url(#pgArchGrad)"
          strokeWidth="8"
          strokeLinecap="round"
          strokeLinejoin="round"
        />

        {/* 
          2. The Upick-Style Verification Swoop:
          Flows cleanly from bottom-left through the curve and accelerates upward 
          into a crisp, sharp verification checkmark
        */}
        <path
          d="M 20 54
             C 26 66, 38 78, 50 78
             C 62 78, 70 70, 78 54
             C 84 42, 86 28, 86 20
             C 83 26, 76 34, 69 41
             C 61 49, 54 55, 47 55
             C 41 55, 34 50, 30 46
             Z"
          fill="url(#pgSwoopGrad)"
        />

        {/* 3. Verification Accent Node */}
        <circle cx="48" cy="34" r="4" fill="#00D26A" />
      </svg>

      {showText && (
        <div className="flex items-center leading-none">
          <span className={`text-base font-bold tracking-tight ${theme === 'dark' ? 'text-white' : 'text-[#0c0f0d]'}`}>
            authentic
          </span>
          <span className="text-base font-bold text-[#00D26A] ml-0.5">
            -et
          </span>
        </div>
      )}
    </div>
  );
};
