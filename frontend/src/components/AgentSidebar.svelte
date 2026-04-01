<script>
  import { settings } from '../stores/index.js';
  import { t } from '../i18n/index.js';

  let activePrompt = null;
  let copiedPrompt = '';

  $: sageUrl = typeof window !== 'undefined' ? window.location.origin : '';
  $: apiToken = $settings?.api_token || '';

  const agentFeatures = [
    {
      id: 'screenshot',
      icon: '📸',
      title: '截图录入',
      desc: '截图发给 Agent，自动识别并录入订阅',
    },
    {
      id: 'reminder',
      icon: '⏰',
      title: '到期提醒',
      desc: '让 Agent 每天检查并发送到期提醒',
    },
    {
      id: 'report',
      icon: '📊',
      title: '月度报告',
      desc: '让 Agent 每月自动生成消费分析报告',
    },
  ];

  function getPrompt(featureId) {
    const url = typeof window !== 'undefined' ? window.location.origin : sageUrl;
    const prompts = {
      screenshot: `你是我的订阅管家。当我发送订阅相关的截图时（账单、邮件、App Store 收据等），识别其中的订阅信息并录入 Sage。

识别后调用 POST ${url}/api/agent/subs 创建记录。
Header: X-API-Token: ${apiToken}

请求体示例：
{
  "name": "服务名",
  "category": "ai",
  "price": 9.99,
  "currency": "USD",
  "cycle": "monthly",
  "start_date": "2026-01-01",
  "next_renewal": "2026-02-01"
}

分类可选: ai, saas, media, domain, vps, app, vpn, dev, membership, gaming, edu, storage, other
周期可选: monthly, yearly, quarterly, weekly, lifetime
币种可选: USD, CNY, EUR, GBP, JPY, HKD, TWD, KRW

识别后先确认信息，我同意后再录入。`,

      reminder: `你是我的订阅管家。每天早上 9:00 检查我的订阅到期情况。

步骤：
1. GET ${url}/api/agent/stats/upcoming?days=7
2. Header: X-API-Token: ${apiToken}
3. 如果有即将到期的订阅，用简洁格式提醒我：
   - 服务名 | 分类 | 到期日期 | 还有 X 天 | 金额 币种
   - 按紧急程度排序（最紧急的在前）
4. 如果没有即将到期的订阅，不发消息。

Sage 地址：${url}
API Token：${apiToken}`,

      report: `你是我的订阅管家。每月 1 号生成一份上月的订阅消费报告。

步骤：
1. GET ${url}/api/agent/stats/overview — 获取总览数据
2. GET ${url}/api/agent/stats/by-category — 获取分类数据
3. GET ${url}/api/agent/subs?status=active — 获取活跃订阅列表

Header: X-API-Token: ${apiToken}

报告格式：
📊 Sage 月度订阅报告

💰 总支出：月均 X / 年均 Y（基准货币）
📈 环比变化：比上月 +X%/-X%
🏷️ 分类明细：（按金额排序）
   - AI 服务：X/月（N 项）
   - ...
⚠️ 即将到期：（7天内）
   - 服务名 - X天后到期
💡 建议：
   - 有 N 个订阅 90 天未使用，考虑取消
   - 有 N 个服务存在功能重叠

Sage 地址：${url}
API Token：${apiToken}`,
    };
    return prompts[featureId] || '';
  }

  async function copyFeaturePrompt(featureId) {
    const text = getPrompt(featureId);
    try {
      await navigator.clipboard.writeText(text);
      copiedPrompt = featureId;
      setTimeout(() => copiedPrompt = '', 2000);
    } catch (_) {}
  }

  function togglePrompt(featureId) {
    activePrompt = activePrompt === featureId ? null : featureId;
  }

  async function copyConfig() {
    const url = typeof window !== 'undefined' ? window.location.origin : '';
    const text = `Sage 订阅管理工具 API：地址 ${url}，Token ${apiToken}。用 X-API-Token header 认证。可用接口：GET /api/agent/subs（列表，支持 ?category=&status= 筛选）、POST /api/agent/subs（新增）、PUT /api/agent/subs/:id（更新）、DELETE /api/agent/subs/:id（删除）、GET /api/agent/stats/overview（总览）、GET /api/agent/stats/by-category（分类统计）、GET /api/agent/stats/upcoming?days=N（即将到期）。分类: ai/saas/media/domain/vps/app/vpn/dev/membership/gaming/edu/storage/other。周期: monthly/yearly/quarterly/weekly/lifetime。`;
    try {
      await navigator.clipboard.writeText(text);
      copiedPrompt = 'config';
      setTimeout(() => copiedPrompt = '', 2000);
    } catch (_) {}
  }
</script>

<div class="ai-card">
  <div class="ai-header">
    <span class="ai-title">{$t('agent_sidebar.title')}</span>
  </div>
  <p class="ai-hint">{$t('agent_sidebar.not_connected_desc')}</p>
  <div class="ai-token-row">
    <code class="ai-token">{apiToken ? apiToken.slice(0, 10) + '...' + apiToken.slice(-6) : '...'}</code>
    <button class="btn-copy-config" class:copied={copiedPrompt === 'config'} on:click={copyConfig} title="复制完整接入配置">
      {copiedPrompt === 'config' ? '✓ Copied' : '📋 Copy Config'}
    </button>
  </div>
  <div class="ai-features">
    {#each agentFeatures as feat}
      <div class="ai-feat-card" class:active={activePrompt === feat.id}>
        <button class="ai-feat-toggle" on:click={() => togglePrompt(feat.id)}>
          <span class="ai-feat-left">
            <span class="ai-feat-icon">{feat.icon}</span>
            <span class="ai-feat-info">
              <span class="ai-feat-name">{feat.title}</span>
              <span class="ai-feat-desc">{feat.desc}</span>
            </span>
          </span>
          <span class="ai-feat-expand">{activePrompt === feat.id ? '▴' : '▾'}</span>
        </button>
        {#if activePrompt === feat.id}
          <div class="ai-prompt-box">
            <pre class="ai-prompt-text">{getPrompt(feat.id)}</pre>
            <button class="btn-copy-prompt" on:click={() => copyFeaturePrompt(feat.id)}>
              {copiedPrompt === feat.id ? '✓ Copied' : '📋 Copy'}
            </button>
          </div>
        {/if}
      </div>
    {/each}
  </div>
</div>

<style>
  .ai-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-left: 3px solid var(--primary);
    border-radius: var(--radius);
    padding: 20px;
  }

  .ai-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
  }

  .ai-title { font-size: 15px; font-weight: 600; }

  .ai-hint {
    font-size: 12px;
    color: var(--text-secondary);
    line-height: 1.4;
    margin-bottom: 12px;
  }

  .ai-token-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
  }

  .ai-token {
    flex: 1;
    font-family: 'DM Sans', monospace;
    font-size: 11px;
    color: var(--text-secondary);
    background: var(--card);
    padding: 6px 10px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    user-select: all;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .btn-copy-config {
    padding: 6px 8px;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    font-size: 12px;
    cursor: pointer;
    transition: all var(--transition);
    flex-shrink: 0;
    color: var(--text-secondary);
  }

  .btn-copy-config:hover { background: var(--hover); border-color: var(--primary); }
  .btn-copy-config.copied { color: var(--success); border-color: var(--success); background: rgba(74, 222, 128, 0.08); }

  .ai-features { display: flex; flex-direction: column; gap: 10px; }

  .ai-feat-card {
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    overflow: hidden;
    transition: border-color var(--transition);
  }

  .ai-feat-card.active { border-color: var(--primary); }

  .ai-feat-toggle {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 14px;
    background: var(--card);
    border: none;
    cursor: pointer;
    transition: background var(--transition);
    color: var(--text-primary);
  }

  .ai-feat-toggle:hover { background: var(--hover); }
  .ai-feat-left { display: flex; align-items: center; gap: 10px; min-width: 0; }
  .ai-feat-icon { font-size: 18px; flex-shrink: 0; }
  .ai-feat-info { display: flex; flex-direction: column; gap: 1px; text-align: left; min-width: 0; }
  .ai-feat-name { font-size: 13px; font-weight: 600; }
  .ai-feat-desc { font-size: 11px; color: var(--text-secondary); }
  .ai-feat-expand { font-size: 24px; font-weight: 700; color: var(--text-tertiary); flex-shrink: 0; margin-left: 8px; line-height: 1; }

  .ai-prompt-box {
    padding: 12px;
    border-top: 1px solid var(--border);
    background: var(--bg);
    animation: fadeIn 0.2s ease;
  }

  .ai-prompt-text {
    font-family: 'DM Sans', monospace;
    font-size: 11px;
    color: var(--text-secondary);
    white-space: pre-wrap;
    word-break: break-all;
    line-height: 1.5;
    margin: 0 0 10px 0;
    max-height: 200px;
    overflow-y: auto;
  }

  .btn-copy-prompt {
    padding: 6px 12px;
    background: var(--primary);
    color: white;
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-weight: 500;
    transition: background var(--transition);
    width: 100%;
  }

  .btn-copy-prompt:hover { background: var(--primary-light); }

  @media (max-width: 1100px) {
    .ai-features { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
    .ai-feat-desc { white-space: normal; }
  }

  @media (max-width: 768px) {
    .ai-features { grid-template-columns: 1fr; }
  }
</style>
