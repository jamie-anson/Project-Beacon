/**** Tailwind config for Beacon Portal ****/
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{js,jsx}',
  ],
  theme: {
    extend: {
      animation: {
        shimmer: 'shimmer 2s infinite',
      },
      keyframes: {
        shimmer: {
          '0%': { transform: 'translateX(-100%)' },
          '100%': { transform: 'translateX(100%)' },
        },
      },
      colors: {
        beacon: {
          50: '#fef7f0',   // light peach tint
          400: '#fab387',  // catppuccin peach (primary accent)
          600: '#f38ba8',  // catppuccin pink (secondary accent)
        },
        // Catppuccin Mocha colors for dark theme
        ctp: {
          base: '#1e1e2e',
          mantle: '#181825',
          crust: '#11111b',
          text: '#cdd6f4',
          subtext1: '#bac2de',
          subtext0: '#a6adc8',
          overlay2: '#9399b2',
          overlay1: '#7f849c',
          overlay0: '#6c7086',
          surface2: '#585b70',
          surface1: '#45475a',
          surface0: '#313244',
          blue: '#89b4fa',
          lavender: '#b4befe',
          sapphire: '#74c7ec',
          sky: '#89dceb',
          teal: '#94e2d5',
          green: '#a6e3a1',
          yellow: '#f9e2af',
          peach: '#fab387',
          maroon: '#eba0ac',
          red: '#f38ba8',
          mauve: '#cba6f7',
          pink: '#f5c2e7',
          flamingo: '#f2cdcd',
          rosewater: '#f5e0dc',
        },
      },
    },
  },
  plugins: [
    // Catppuccin plugin - optional to prevent build failures
    (() => {
      try {
        return require("@catppuccin/tailwindcss")({
          prefix: "ctp",
          defaultFlavour: "mocha",
        });
      } catch (e) {
        console.warn('Catppuccin Tailwind plugin not available, skipping...');
        return null;
      }
    })(),
  ].filter(Boolean),
};
