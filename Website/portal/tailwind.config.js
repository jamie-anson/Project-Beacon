/**** Tailwind config for Beacon Portal ****/
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{js,jsx}',
  ],
  theme: {
    extend: {
      colors: {
        beacon: {
          400: '#b15e39', // warm accent
          600: '#55250d', // deep brown
        },
      },
    },
  },
  plugins: [],
};
