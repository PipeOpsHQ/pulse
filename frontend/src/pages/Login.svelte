<script>
  import { navigate } from "../lib/router";
  import { login } from "../stores/auth";

  let email = "";
  let password = "";
  let mfaCode = "";
  let error = "";
  let loading = false;
  let mfaRequired = false;
  let tempUserId = "";

  async function handleLogin(e) {
    if (e) e.preventDefault();
    loading = true;
    error = "";

    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      const data = await res.json();
      if (res.ok) {
        if (data.mfa_required) {
          mfaRequired = true;
          tempUserId = data.user_id;
        } else {
          login(data.token, data.user);
          navigate("/", { replace: true });
        }
      } else {
        error = data.message || "Invalid credentials";
      }
    } catch (e) {
      error = "Something went wrong";
    } finally {
      loading = false;
    }
  }

  async function handleMFAVerify(e) {
    if (e) e.preventDefault();
    loading = true;
    error = "";

    try {
      const res = await fetch("/api/auth/mfa/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: tempUserId, code: mfaCode }),
      });

      const data = await res.json();
      if (res.ok) {
        login(data.token, data.user);
        navigate("/", { replace: true });
      } else {
        error = "Invalid verification code";
      }
    } catch (e) {
      error = "MFA verification failed";
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-page">
  <div class="background-mesh"></div>

  <div class="login-container">
    <div class="brand">
      <div class="logo-icon">P</div>
      <h1>Pulse</h1>
    </div>

    <div class="login-card">
      {#if !mfaRequired}
        <form onsubmit={handleLogin}>
          <h2>Welcome Back</h2>
          <p class="subtitle">Enter your credentials to access the dashboard</p>

          {#if error}
            <div class="error-alert">{error}</div>
          {/if}

          <div class="form-group">
            <label for="email">Email</label>
            <input
              type="email"
              id="email"
              bind:value={email}
              placeholder="admin@example.com"
              required
            />
          </div>

          <div class="form-group">
            <label for="password">Password</label>
            <input
              type="password"
              id="password"
              bind:value={password}
              placeholder="••••••••"
              required
            />
          </div>

          <button type="submit" class="submit-btn" disabled={loading}>
            {#if loading}
              <span class="spinner"></span>
            {:else}
              Sign In
            {/if}
          </button>
        </form>
      {:else}
        <form onsubmit={handleMFAVerify}>
          <h2>Security Verification</h2>
          <p class="subtitle">
            Enter the 6-digit code from your authenticator app
          </p>

          {#if error}
            <div class="error-alert">{error}</div>
          {/if}

          <div class="form-group">
            <label for="mfaCode">Verification Code</label>
            <input
              type="text"
              id="mfaCode"
              bind:value={mfaCode}
              placeholder="000000"
              maxlength="6"
              required
              class="mfa-input"
            />
          </div>

          <button type="submit" class="submit-btn" disabled={loading}>
            {#if loading}
              <span class="spinner"></span>
            {:else}
              Verify & Sign In
            {/if}
          </button>

          <button
            type="button"
            class="back-btn"
            onclick={() => (mfaRequired = false)}
          >
            Back to Login
          </button>
        </form>
      {/if}
    </div>
  </div>
</div>

<style>
  .login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: #050505;
    position: relative;
    overflow: hidden;
    color: #fff;
  }

  .background-mesh {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background:
      radial-gradient(
        circle at 15% 50%,
        rgba(99, 102, 241, 0.15) 0%,
        transparent 25%
      ),
      radial-gradient(
        circle at 85% 30%,
        rgba(168, 85, 247, 0.15) 0%,
        transparent 25%
      );
    z-index: 1;
  }

  .login-container {
    position: relative;
    z-index: 2;
    width: 100%;
    max-width: 480px;
    padding: 2rem;
    display: flex;
    flex-direction: column;
    gap: 2rem;
  }

  .brand {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
  }

  .logo-icon {
    width: 40px;
    height: 40px;
    background: linear-gradient(135deg, #6366f1, #a855f7);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
    font-size: 1.25rem;
    box-shadow: 0 0 20px rgba(99, 102, 241, 0.5);
  }

  h1 {
    font-size: 1.5rem;
    font-weight: 700;
    margin: 0;
    background: linear-gradient(to right, #fff, #94a3b8);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .login-card {
    background: rgba(255, 255, 255, 0.03);
    backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.05);
    padding: 2.5rem;
    border-radius: 16px;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
  }

  h2 {
    margin: 0 0 0.5rem 0;
    font-size: 1.25rem;
    font-weight: 600;
    text-align: center;
  }

  .subtitle {
    margin: 0 0 2rem 0;
    color: #94a3b8;
    text-align: center;
    font-size: 0.875rem;
  }

  .form-group {
    margin-bottom: 1.5rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    color: #cbd5e1;
    font-size: 0.875rem;
    font-weight: 500;
  }

  input {
    width: 100%;
    padding: 0.75rem 1rem;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: #fff;
    font-size: 0.9375rem;
    transition: all 0.2s;
    box-sizing: border-box; /* Fix for width overflow */
  }

  input:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
    background: rgba(0, 0, 0, 0.5);
  }

  .submit-btn {
    width: 100%;
    padding: 0.875rem;
    background: linear-gradient(135deg, #6366f1, #8b5cf6);
    color: white;
    border: none;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    transition:
      transform 0.1s,
      opacity 0.2s;
    margin-top: 1rem;
  }

  .submit-btn:hover {
    opacity: 0.9;
    transform: translateY(-1px);
  }

  .submit-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .error-alert {
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    color: #fca5a5;
    padding: 0.75rem;
    border-radius: 8px;
    font-size: 0.875rem;
    text-align: center;
    margin-bottom: 1.5rem;
  }

  .back-btn {
    width: 100%;
    margin-top: 1rem;
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #94a3b8;
    padding: 0.75rem;
    border-radius: 8px;
    font-size: 0.875rem;
    cursor: pointer;
    transition: all 0.2s;
  }

  .back-btn:hover {
    background: rgba(255, 255, 255, 0.05);
    color: #fff;
  }

  .mfa-input {
    text-align: center;
    font-size: 1.5rem;
    letter-spacing: 0.5rem;
    font-weight: 700;
  }

  .spinner {
    display: inline-block;
    width: 1.25rem;
    height: 1.25rem;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-radius: 50%;
    border-top-color: #fff;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
