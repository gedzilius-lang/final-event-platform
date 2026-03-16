import type { Config } from 'tailwindcss'

const config: Config = {
  content: ['./src/**/*.{js,ts,jsx,tsx,mdx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50:  '#f5f3ff',
          100: '#ede9fe',
          500: '#6C2BD9',
          600: '#5b21b6',
          700: '#4c1d95',
        },
      },
    },
  },
  plugins: [],
}
export default config
