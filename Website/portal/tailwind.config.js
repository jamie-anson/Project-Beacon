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
          50: '#fef7f0',   // light peach tint
          400: '#fab387',  // catppuccin peach (primary accent)
          600: '#f38ba8',  // catppuccin pink (secondary accent)
        },
      },
    },
  },
  plugins: [
    require("@catppuccin/tailwindcss")({
      prefix: "ctp",
      defaultFlavour: "mocha",
    }),
  ],
};
