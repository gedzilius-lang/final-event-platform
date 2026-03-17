import type { Config } from 'tailwindcss'

const config: Config = {
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './lib/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        // NiteOS brand — dark-first, amber accent
        nite: {
          bg: '#0a0a0a',
          surface: '#141414',
          border: '#242424',
          accent: '#f59e0b',   // amber-500
          accent2: '#6366f1',  // indigo-500 (secondary)
          text: '#f5f5f5',
          muted: '#737373',
        },
      },
    },
  },
  plugins: [],
}

export default config
