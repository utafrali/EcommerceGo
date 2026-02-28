import type { Config } from 'tailwindcss';

const config: Config = {
  content: [
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/contexts/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: '#d63384',
          light: '#e879aa',
          lighter: '#fce7f3',
          dark: '#9d174d',
          accent: '#f97316',
          'accent-light': '#fff7ed',
        },
      },
    },
  },
  plugins: [],
};

export default config;
