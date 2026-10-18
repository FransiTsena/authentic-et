/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', 'Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        display: ['"Plus Jakarta Sans"', 'Inter', 'sans-serif'],
      },
      colors: {
        ink: '#0c0f0d',
        orange: {
          DEFAULT: '#ff6b00',
          50: '#fff7ed',
          100: '#ffedd5',
          200: '#fed7aa',
          500: '#ff6b00',
          600: '#ea580c',
          700: '#c2410c',
        },
        green: {
          DEFAULT: '#079447',
          50: '#f0fdf4',
          100: '#dcfce7',
          500: '#079447',
          600: '#16a34a',
        },
        brand: {
          50: '#eff6ff',
          100: '#dbeafe',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
        },
      },
      boxShadow: {
        lift: '0 0 0 1px rgba(12,15,13,0.08), 0 20px 40px -12px rgba(12,15,13,0.18)',
        card: '0 0 0 1px rgba(12,15,13,0.06), 0 2px 4px rgba(12,15,13,0.04), 0 24px 64px -12px rgba(12,15,13,0.12)',
      },
    },
  },
  plugins: [],
};
