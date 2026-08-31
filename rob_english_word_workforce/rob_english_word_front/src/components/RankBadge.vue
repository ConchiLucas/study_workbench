<template>
  <div class="rank-badge" :class="tier.className">
    <div class="badge-icon-wrapper">
      <div class="badge-glow"></div>
      <div class="badge-shape">
        <span class="badge-symbol">{{ tier.symbol }}</span>
      </div>
      <!-- 王者专属装饰 -->
      <div v-if="tier.className === 'tier-challenger'" class="crown-decoration">👑</div>
    </div>
    <div class="badge-info">
      <div class="tier-name">{{ tier.name }}</div>
      <div class="level-text">Lv . {{ level }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  level: number
}>()

const tier = computed(() => {
  const l = props.level || 1
  if (l <= 10) return { name: '倔强青铜', className: 'tier-bronze', symbol: 'Ⅲ' }
  if (l <= 20) return { name: '秩序白银', className: 'tier-silver', symbol: 'Ⅱ' }
  if (l <= 30) return { name: '荣耀黄金', className: 'tier-gold', symbol: 'Ⅰ' }
  if (l <= 40) return { name: '尊贵铂金', className: 'tier-platinum', symbol: '✧' }
  if (l <= 50) return { name: '永恒钻石', className: 'tier-diamond', symbol: '💎' }
  if (l <= 60) return { name: '至尊星耀', className: 'tier-master', symbol: '★' }
  return { name: '最强王者', className: 'tier-challenger', symbol: '♆' }
})
</script>

<style scoped>
.rank-badge {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  position: relative;
  z-index: 1;
  padding: 10px;
}

.badge-icon-wrapper {
  position: relative;
  width: 70px;
  height: 70px;
  display: flex;
  justify-content: center;
  align-items: center;
  transition: transform 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.rank-badge:hover .badge-icon-wrapper {
  transform: scale(1.15) translateY(-5px);
}

.badge-glow {
  position: absolute;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  filter: blur(15px);
  z-index: -1;
  opacity: 0.6;
  animation: pulse 2s infinite alternate ease-in-out;
}

.badge-shape {
  width: 60px;
  height: 68px;
  /* 默认为六边形 */
  clip-path: polygon(50% 0%, 100% 25%, 100% 75%, 50% 100%, 0% 75%, 0% 25%);
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 28px;
  color: white;
  text-shadow: 0 2px 4px rgba(0,0,0,0.5);
  position: relative;
  box-shadow: inset 0 0 15px rgba(255,255,255,0.5);
  transition: all 0.3s ease;
}

/* 顶部高光反射，增加金属质感 */
.badge-shape::after {
  content: '';
  position: absolute;
  top: 0; left: 0; width: 100%; height: 50%;
  background: linear-gradient(to bottom, rgba(255,255,255,0.6), transparent);
  pointer-events: none;
}

.crown-decoration {
  position: absolute;
  top: -20px;
  font-size: 32px;
  filter: drop-shadow(0 0 10px #ffd700);
  animation: float 3s ease-in-out infinite;
  z-index: 2;
}

.badge-info {
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tier-name {
  font-size: 16px;
  font-weight: 900;
  letter-spacing: 1.5px;
  text-transform: uppercase;
}

.level-text {
  font-size: 13px;
  font-weight: 800;
  color: rgba(255, 255, 255, 0.7);
  background: rgba(0,0,0,0.3);
  padding: 2px 10px;
  border-radius: 10px;
  border: 1px solid rgba(255,255,255,0.1);
}

/* 青铜 Bronze */
.tier-bronze .badge-shape { background: linear-gradient(135deg, #cd7f32, #6b3e10); border: 2px solid #e29c5b; }
.tier-bronze .badge-glow { background: #cd7f32; }
.tier-bronze .tier-name { color: #cd7f32; text-shadow: 0 0 5px rgba(205,127,50,0.5); }

/* 白银 Silver */
.tier-silver .badge-shape { background: linear-gradient(135deg, #e0e0e0, #757575); border: 2px solid #ffffff; }
.tier-silver .badge-glow { background: #e0e0e0; }
.tier-silver .tier-name { color: #e0e0e0; text-shadow: 0 0 5px rgba(224,224,224,0.5); }

/* 黄金 Gold */
.tier-gold .badge-shape { background: linear-gradient(135deg, #ffe066, #f9a826); border: 2px solid #fff5b3; box-shadow: inset 0 0 20px rgba(255,255,255,0.8); }
.tier-gold .badge-glow { background: #ffd700; opacity: 0.8;}
.tier-gold .tier-name { color: #ffd700; text-shadow: 0 0 10px rgba(255,215,0,0.6); }

/* 铂金 Platinum */
.tier-platinum .badge-shape { background: linear-gradient(135deg, #00f2fe, #4facfe); border: 2px solid #b3f6ff; clip-path: polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%); width: 66px; height: 66px;}
.tier-platinum .badge-glow { background: #00f2fe; }
.tier-platinum .tier-name { color: #00f2fe; text-shadow: 0 0 10px rgba(0,242,254,0.6); }

/* 钻石 Diamond */
.tier-diamond .badge-shape { background: linear-gradient(135deg, #ccffff, #0052d4); border: 2px solid #ffffff; clip-path: polygon(50% 0%, 100% 25%, 50% 100%, 0% 25%); width: 68px; height: 68px;}
.tier-diamond .badge-glow { background: #0088ff; opacity: 1; filter: blur(20px);}
.tier-diamond .tier-name { color: #b9f2ff; text-shadow: 0 0 15px rgba(0,82,212,0.8); }

/* 星耀 Master */
.tier-master .badge-shape { background: linear-gradient(135deg, #e0c3fc, #8ec5fc); border: 2px solid #ffffff; clip-path: polygon(50% 0%, 100% 38%, 82% 100%, 18% 100%, 0% 38%); width: 68px; height: 66px; }
.tier-master .badge-glow { background: #e0c3fc; opacity: 1; filter: blur(20px);}
.tier-master .tier-name { color: #e0c3fc; text-shadow: 0 0 15px rgba(224,195,252,0.8); }

/* 王者 Challenger */
.tier-challenger .badge-shape { background: linear-gradient(135deg, #ff0844, #ffb199); border: 3px solid #ffd700; clip-path: polygon(50% 0%, 100% 20%, 100% 80%, 50% 100%, 0% 80%, 0% 20%); width: 72px; height: 80px; box-shadow: inset 0 0 25px rgba(255,215,0,0.8); }
.tier-challenger .badge-glow { background: #ff0844; filter: blur(25px); opacity: 1; }
.tier-challenger .tier-name { 
  background: linear-gradient(to right, #ff0844, #ffd700); 
  -webkit-background-clip: text; 
  background-clip: text;
  -webkit-text-fill-color: transparent; 
  text-shadow: 0 0 25px rgba(255,8,68,0.6); 
}

@keyframes pulse {
  from { transform: scale(0.9); opacity: 0.5; }
  to { transform: scale(1.15); opacity: 0.85; }
}

@keyframes float {
  0% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
  100% { transform: translateY(0); }
}
</style>
