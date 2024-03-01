/** @type {import('tailwindcss').Config} */
/* eslint-env node */
// TODO: upgrade to ESM
module.exports = {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        tracoor: ['tracoor'],
      },
      boxShadow: {
        'inner-lg': '0px 0px 5px 1px rgba(0,0,0,0.5) inset',
        'inner-xl': '0px 0px 10px 2px rgba(0,0,0,0.5) inset',
      },
      keyframes: {
        'pulse-light': {
          '50%': { opacity: 0.97 },
        },
      },
      animation: {
        'pulse-light': 'pulse-light 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      screens: {
        sm: '640px',
        md: '768px',
        lg: '1024px',
        xl: '1280px',
        '2xl': '1536px',
        '3xl': '1920px',
        '4xl': '2560px',
      },
    },
  },
  plugins: [],
  darkMode: 'class',
};
