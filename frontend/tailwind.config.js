/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        // Palette เดิมของโปรเจกต์ — "kraft market & price tag"
        bg: '#f6f4ee',
        surface: '#ffffff',
        ink: {
          DEFAULT: '#22252a',
          soft: 'rgba(34,37,42,0.62)',
          faint: 'rgba(34,37,42,0.4)',
        },
        line: 'rgba(34,37,42,0.13)',
        orange: {
          DEFAULT: '#e2622a',
          dark: '#c94f1c',
        },
        green: '#3d8361', // trust / สถานะ available
        amber: '#d69a1f', // สถานะ reserved
        red: '#c84a3c', // error / sold
      },
      fontFamily: {
        display: ['Kanit', 'Noto Sans Thai', 'sans-serif'],
        body: ['Sarabun', 'Noto Sans Thai', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      boxShadow: {
        card: '0 10px 28px rgba(34,37,42,0.08)',
      },
      maxWidth: {
        container: '1080px',
      },
    },
  },
  plugins: [],
}
