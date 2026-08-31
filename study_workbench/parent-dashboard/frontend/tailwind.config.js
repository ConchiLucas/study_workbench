/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: { 50: '#FFF5F8', 100: '#FFE4EE', 300: '#FBA9C5', 500: '#F472A6', 700: '#DB2777' },
        // 答题端用一套独立的糖果色，让孩子一眼知道"这是我的地方"，
        // 而不是家长看板的延伸。
        candy: {
          cream: '#FFF6EC',
          paper: '#FFFDF9',
          ink: '#3D3230',
          mute: '#9C8C86',
          peach: '#FFB4A2',
          mint: '#9BE6C4',
          lemon: '#FFDE8A',
          sky: '#9FD4F5',
          go: '#FF8A5B',
          right: '#3FC77F',
          wrong: '#FB7185',
        },
      },
      borderRadius: { xl2: '1.25rem', kid: '2rem' },
      fontFamily: {
        // ui-rounded 在 iPad Safari 上直接拿到系统的 SF Pro Rounded，
        // 汉字回落到苹方。圆头字形对这个年龄更友好，而且不用加载网络字体
        // ——Google Fonts 在国内不可靠，家里断网时不能让题目变成方框。
        kid: ['ui-rounded', '"SF Pro Rounded"', '-apple-system', '"PingFang SC"', 'sans-serif'],
      },
      boxShadow: {
        // 无模糊的偏移阴影：按下时阴影收缩 + 元素下移，做出实体按键的手感。
        // 5 岁孩子需要明确的"我按到了"反馈。
        sticker: '0 6px 0 rgba(61,50,48,0.18)',
        'sticker-sm': '0 3px 0 rgba(61,50,48,0.18)',
        pop: '0 10px 0 rgba(61,50,48,0.15)',
      },
      keyframes: {
        shake: {
          '0%,100%': { transform: 'translateX(0)' },
          '20%,60%': { transform: 'translateX(-9px)' },
          '40%,80%': { transform: 'translateX(9px)' },
        },
        popIn: {
          '0%': { transform: 'scale(0.4)', opacity: '0' },
          '70%': { transform: 'scale(1.12)', opacity: '1' },
          '100%': { transform: 'scale(1)', opacity: '1' },
        },
        floatUp: {
          '0%': { transform: 'translateY(12px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        bob: {
          '0%,100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-8px)' },
        },
      },
      animation: {
        shake: 'shake 0.45s ease-in-out',
        popIn: 'popIn 0.45s cubic-bezier(0.34,1.56,0.64,1) both',
        floatUp: 'floatUp 0.35s ease-out both',
        bob: 'bob 2.4s ease-in-out infinite',
      },
    },
  },
  plugins: [],
}
