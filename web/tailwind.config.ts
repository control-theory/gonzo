import type { Config } from 'tailwindcss'

const config: Config = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Dstl8 severity colors
        severity: {
          fatal: '#b91c1c',
          error: '#dc2626',
          warn: '#f97316',
          info: '#3b82f6',
          debug: '#6b7280',
          trace: '#10b981',
          unknown: '#9ca3af',
        },
        // Dstl8 brand
        brand: {
          primary: '#6366f1',
          secondary: '#8b5cf6',
        },
        // Dashboard surface colors
        surface: {
          DEFAULT: '#ffffff',
          secondary: '#f8fafc',
          dark: '#0f172a',
          'dark-secondary': '#1e293b',
        },
      },
    },
  },
  plugins: [],
}
export default config
