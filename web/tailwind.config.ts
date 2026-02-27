import type { Config } from 'tailwindcss';

const config: Config = {
  content: [
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: '#b5705b',   // warm rose-gold primary
          light: '#c98a77',     // lighter rose-gold
          lighter: '#f5e1da',   // very light rose-gold tint
          dark: '#8f5443',      // deeper rose-gold for hover states
          accent: '#c8873e',    // warm amber accent
          'accent-light': '#fdf3e7', // light amber tint
        },
      },
      fontFamily: {
        sans: [
          'var(--font-inter)',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'sans-serif',
        ],
      },
      keyframes: {
        shimmer: {
          '0%': { transform: 'translateX(-100%)' },
          '100%': { transform: 'translateX(100%)' },
        },
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideInLeft: {
          '0%': { opacity: '0', transform: 'translateX(-100%)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(100%)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
      },
      animation: {
        shimmer: 'shimmer 3s ease-in-out infinite',
        'fade-in': 'fadeIn 200ms ease-out',
        'slide-up': 'slideUp 250ms ease-out',
        'slide-in-left': 'slideInLeft 300ms ease-out',
        'slide-in-right': 'slideInRight 300ms ease-out',
      },
    },
  },
  plugins: [],
};

export default config;
