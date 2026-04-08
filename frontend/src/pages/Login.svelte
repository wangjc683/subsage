<script>
  import { login, setup, getAuthStatus } from '../api/index.js';
  import { t } from '../i18n/index.js';
  import { APP_VERSION } from '../version.js';



  let username = '';
  let password = '';
  let confirmPassword = '';
  let loading = false;
  let error = '';
  let initialized = false;

  async function checkStatus() {
    try {
      const data = await getAuthStatus();
      initialized = data.initialized;
    } catch (e) {
      initialized = false;
    }
  }

  checkStatus();

  async function handleSubmit() {
    error = '';
    loading = true;

    try {
      if (!initialized) {
        if (password !== confirmPassword) {
          error = $t('login.password_mismatch');
          loading = false;
          return;
        }
        const data = await setup(username, password);
        onLogin(data);
      } else {
        const data = await login(username, password);
        onLogin(data);
      }
    } catch (e) {
      error = e.message || $t('login.failed');
    } finally {
      loading = false;
    }
  }

  function onLogin(data) {
    if (data.token) {
      import('../stores/index.js').then(({ auth }) => {
        auth.login(data.token, data.username);
        window.location.hash = '#/overview';
      });
    }
  }
</script>

<div class="login-wrapper">
  <!-- Left: Brand Panel -->
  <div class="brand-panel">
    <div class="brand-bg-pattern"></div>
    <div class="brand-content">
      <div class="brand-logo">
        <svg viewBox="0 0 32 32" width="44" height="44">
          <path d="M5 0h22c2.8 0 5 2.2 5 5v3c0 2.8-2.2 5-5 5h-9c-2.2 0-4 1.8-4 4H5c-2.8 0-5-2.2-5-5V5c0-2.8 2.2-5 5-5z" fill="white" fill-opacity="0.9"/>
          <path d="M27 32H5c-2.8 0-5-2.2-5-5v-3c0-2.8 2.2-5 5-5h9c2.2 0 4-1.8 4-4h9c2.8 0 5 2.2 5 5v7c0 2.8-2.2 5-5 5z" fill="white" fill-opacity="0.9"/>
        </svg>
      </div>
      <h1 class="brand-title">SubSage</h1>
      <p class="brand-slogan">{$t('login.tagline')}</p>

      <div class="brand-features">
        <div class="feature-item">
          <span class="feature-icon">🤖</span>
          <div class="feature-text">
            <span class="feature-name">{$t('login.feature1_title')}</span>
            <span class="feature-desc">{$t('login.feature1_desc')}</span>
          </div>
        </div>
        <div class="feature-item">
          <span class="feature-icon">💬</span>
          <div class="feature-text">
            <span class="feature-name">{$t('login.feature2_title')}</span>
            <span class="feature-desc">{$t('login.feature2_desc')}</span>
          </div>
        </div>
        <div class="feature-item">
          <span class="feature-icon">💬</span>
          <div class="feature-text">
            <span class="feature-name">{$t('login.feature3_title')}</span>
            <span class="feature-desc">{$t('login.feature3_desc')}</span>
          </div>
        </div>
        <div class="feature-item">
          <span class="feature-icon">🔒</span>
          <div class="feature-text">
            <span class="feature-name">{$t('settings.license_val')}</span>
            <span class="feature-desc">{$t('login.feature2_desc')}</span>
          </div>
        </div>
      </div>

    </div>
  </div>

  <!-- Right: Login Form -->
  <div class="form-panel">
    <div class="login-card">
      <!-- Mobile brand header (hidden on desktop) -->
      <div class="mobile-brand">
        <div class="mobile-brand-row">
          <svg viewBox="0 0 32 32" width="28" height="28">
            <path d="M5 0h22c2.8 0 5 2.2 5 5v3c0 2.8-2.2 5-5 5h-9c-2.2 0-4 1.8-4 4H5c-2.8 0-5-2.2-5-5V5c0-2.8 2.2-5 5-5z" fill="var(--primary)" fill-opacity="0.85"/>
            <path d="M27 32H5c-2.8 0-5-2.2-5-5v-3c0-2.8 2.2-5 5-5h9c2.2 0 4-1.8 4-4h9c2.8 0 5 2.2 5 5v7c0 2.8-2.2 5-5 5z" fill="var(--primary)" fill-opacity="0.85"/>
          </svg>
          <span class="mobile-brand-name">SubSage</span>
        </div>
        <p class="mobile-brand-slogan">{$t('login.tagline')}</p>
      </div>

      <div class="login-header">
        <h2>{initialized ? $t('login.welcome_back') : $t('login.create_account')}</h2>
        <p class="subtitle">{initialized ? $t('login.welcome_back_desc') : $t('login.create_account_desc')}</p>
      </div>

      <form on:submit|preventDefault={handleSubmit}>
        <div class="field">
          <label for="username">{$t('login.username')}</label>
          <input
            id="username"
            type="text"
            bind:value={username}
            placeholder="Username"
            autocomplete="username"
            required
          />
        </div>

        <div class="field">
          <label for="password">{$t('login.password')}</label>
          <input
            id="password"
            type="password"
            bind:value={password}
            placeholder="Password"
            autocomplete={initialized ? 'current-password' : 'new-password'}
            required
          />
        </div>

        {#if !initialized}
          <div class="field">
            <label for="confirm">{$t('login.confirm_password')}</label>
            <input
              id="confirm"
              type="password"
              bind:value={confirmPassword}
              placeholder="Confirm password"
              autocomplete="new-password"
              required
            />
          </div>
        {/if}

        {#if error}
          <div class="error-msg">{error}</div>
        {/if}

        <button type="submit" class="btn-primary" disabled={loading}>
          {loading ? $t('common.loading') : (initialized ? $t('login.sign_in') : $t('login.start'))}
        </button>
      </form>

      <!-- Mobile features (hidden on desktop) -->
      <div class="mobile-features">
        <div class="mobile-features-divider"></div>
        <div class="mobile-feature-list">
          <div class="mobile-feature"><span>🤖</span><span>{$t('login.mobile_feat1')}</span></div>
          <div class="mobile-feature"><span>📊</span><span>{$t('login.mobile_feat2')}</span></div>
          <div class="mobile-feature"><span>💬</span><span>{$t('login.mobile_feat3')}</span></div>
          <div class="mobile-feature"><span>🔒</span><span>{$t('login.mobile_feat4')}</span></div>
        </div>
      </div>

      <div class="login-footer">
        <span class="login-version">SubSage {APP_VERSION}</span>
      </div>
    </div>
  </div>
</div>

<style>
  .login-wrapper {
    display: flex;
    min-height: 100vh;
    min-height: 100dvh;
  }

  /* ===== Left Brand Panel ===== */
  .brand-panel {
    flex: 1;
    background: linear-gradient(160deg, #2C5A44 0%, #3D7C5F 40%, #4E9B78 70%, #2C5A44 100%);
    background-size: 300% 300%;
    animation: gradientShift 12s ease infinite;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 48px;
    position: relative;
    overflow: hidden;
  }

  /* Geometric line pattern overlay */
  .brand-bg-pattern {
    position: absolute;
    inset: 0;
    background-image:
      radial-gradient(circle at 20% 20%, rgba(255,255,255,0.06) 0%, transparent 50%),
      radial-gradient(circle at 80% 80%, rgba(255,255,255,0.04) 0%, transparent 50%),
      linear-gradient(135deg, transparent 40%, rgba(255,255,255,0.02) 40%, rgba(255,255,255,0.02) 60%, transparent 60%);
    pointer-events: none;
  }

  /* Subtle decorative circles */
  .brand-panel::before {
    content: '';
    position: absolute;
    width: 500px; height: 500px;
    border-radius: 50%;
    border: 1px solid rgba(255,255,255,0.06);
    top: -150px; right: -150px;
  }
  .brand-panel::after {
    content: '';
    position: absolute;
    width: 350px; height: 350px;
    border-radius: 50%;
    border: 1px solid rgba(255,255,255,0.05);
    bottom: -100px; left: -100px;
  }

  .brand-content {
    position: relative;
    z-index: 1;
    max-width: 380px;
    color: white;
  }

  .brand-logo {
    margin-bottom: 20px;
  }

  .brand-logo svg {
    filter: drop-shadow(0 2px 8px rgba(0,0,0,0.15));
  }

  .brand-title {
    font-family: 'DM Sans', sans-serif;
    font-size: 36px;
    font-weight: 700;
    letter-spacing: -0.5px;
    margin-bottom: 8px;
  }

  .brand-slogan {
    font-size: 18px;
    opacity: 0.85;
    margin-bottom: 40px;
    font-weight: 300;
  }

  .brand-features {
    display: flex;
    flex-direction: column;
    gap: 20px;
    margin-bottom: 48px;
  }

  .feature-item {
    display: flex;
    align-items: flex-start;
    gap: 14px;
    transition: transform 0.3s ease;
  }

  .feature-item:hover {
    transform: translateX(4px);
  }

  .feature-icon {
    font-size: 22px;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255,255,255,0.12);
    border-radius: var(--radius);
    flex-shrink: 0;
    transition: all 0.3s ease;
  }

  .feature-item:hover .feature-icon {
    background: rgba(255,255,255,0.18);
    animation: gentleBounce 0.5s ease;
  }

  .feature-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding-top: 2px;
  }

  .feature-name {
    font-size: 15px;
    font-weight: 600;
  }

  .feature-desc {
    font-size: 13px;
    opacity: 0.7;
  }

  /* ===== Right Form Panel ===== */
  .form-panel {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px 32px;
    background: var(--bg);
  }

  .login-card {
    width: 100%;
    max-width: 400px;
    animation: fadeIn 0.5s ease;
  }

  .login-header {
    margin-bottom: 32px;
  }

  .login-header h2 {
    font-size: 24px;
    font-weight: 700;
    margin-bottom: 6px;
    color: var(--text-primary);
  }

  .subtitle {
    color: var(--text-secondary);
    font-size: 14px;
  }

  .field {
    margin-bottom: 18px;
  }

  .field label {
    display: block;
    font-size: 13px;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 6px;
  }

  .field input {
    width: 100%;
    padding: 11px 14px;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: 14px;
    transition: border-color var(--transition), background var(--transition), box-shadow var(--transition);
  }

  .field input:focus {
    border-color: var(--primary);
    background: var(--primary-faint);
    box-shadow: 0 0 0 3px var(--primary-glow);
  }

  .field input::placeholder {
    color: var(--text-tertiary);
  }

  .btn-primary {
    width: 100%;
    padding: 11px 20px;
    background: var(--primary);
    color: white;
    border-radius: var(--radius-sm);
    font-size: 15px;
    font-weight: 500;
    transition: background var(--transition), opacity var(--transition), transform 0.1s ease;
    margin-top: 8px;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--primary-light);
  }

  .btn-primary:active:not(:disabled) {
    transform: scale(0.97);
  }

  .btn-primary:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .error-msg {
    background: rgba(237, 63, 63, 0.1);
    color: var(--error);
    padding: 10px 14px;
    border-radius: var(--radius-sm);
    font-size: 13px;
    margin-bottom: 12px;
    border-left: 3px solid var(--error);
  }

  .login-footer {
    margin-top: 32px;
    text-align: center;
  }

  .login-version {
    font-size: 12px;
    color: var(--text-tertiary);
    letter-spacing: 0.5px;
  }

  /* Mobile-only elements: hidden on desktop */
  .mobile-brand { display: none; }
  .mobile-features { display: none; }

  /* ===== Mobile: single unified column ===== */
  @media (max-width: 768px) {
    .login-wrapper {
      flex-direction: column;
      min-height: 100vh;
      min-height: 100dvh;
      background: var(--bg);
    }

    .brand-panel {
      display: none;
    }

    .mobile-brand {
      display: block;
      margin-bottom: 28px;
    }

    .mobile-brand-row {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 4px;
    }

    .mobile-brand-name {
      font-family: 'DM Sans', sans-serif;
      font-size: 22px;
      font-weight: 700;
      color: var(--text-primary);
      letter-spacing: -0.3px;
    }

    .mobile-brand-slogan {
      font-size: 13px;
      color: var(--text-secondary);
      margin: 0;
    }

    .form-panel {
      flex: 1;
      display: flex;
      flex-direction: column;
      align-items: stretch;
      justify-content: flex-start;
      padding: 40px 24px 24px;
      min-height: 0;
    }

    .login-card {
      max-width: 100%;
      width: 100%;
    }

    .login-header {
      margin-bottom: 24px;
    }

    .login-header h2 {
      font-size: 20px;
    }

    .mobile-features {
      display: block;
      margin-top: 8px;
    }

    .mobile-features-divider {
      height: 1px;
      background: var(--border);
      margin: 24px 0 20px;
    }

    .mobile-feature-list {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 12px 16px;
    }

    .mobile-feature {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 13px;
      color: var(--text-secondary);
    }

    .mobile-feature span:first-child {
      font-size: 16px;
      flex-shrink: 0;
    }

    .login-footer {
      margin-top: 24px;
    }
  }
</style>
