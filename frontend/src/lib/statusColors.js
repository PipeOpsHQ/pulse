/**
 * Status color utilities for consistent color coding across the application
 */

/**
 * Get color classes for HTTP status codes
 */
export function getHttpStatusColor(statusCode) {
  if (statusCode >= 200 && statusCode < 300) {
    return {
      text: 'text-emerald-400',
      bg: 'bg-emerald-500/10',
      border: 'border-emerald-500/20',
      icon: '✅',
      label: 'Success'
    };
  } else if (statusCode >= 300 && statusCode < 400) {
    return {
      text: 'text-cyan-400',
      bg: 'bg-cyan-500/10',
      border: 'border-cyan-500/20',
      icon: '↪️',
      label: 'Redirect'
    };
  } else if (statusCode >= 400 && statusCode < 500) {
    if (statusCode === 401) {
      return {
        text: 'text-red-400',
        bg: 'bg-red-500/10',
        border: 'border-red-500/20',
        icon: '🔒',
        label: 'Unauthorized'
      };
    } else if (statusCode === 403) {
      return {
        text: 'text-red-400',
        bg: 'bg-red-500/10',
        border: 'border-red-500/20',
        icon: '🚫',
        label: 'Forbidden'
      };
    } else if (statusCode === 404) {
      return {
        text: 'text-yellow-400',
        bg: 'bg-yellow-500/10',
        border: 'border-yellow-500/20',
        icon: '🔍',
        label: 'Not Found'
      };
    } else if (statusCode === 429) {
      return {
        text: 'text-orange-400',
        bg: 'bg-orange-500/10',
        border: 'border-orange-500/20',
        icon: '⚠️',
        label: 'Rate Limited'
      };
    } else {
      return {
        text: 'text-yellow-400',
        bg: 'bg-yellow-500/10',
        border: 'border-yellow-500/20',
        icon: '⚠️',
        label: 'Client Error'
      };
    }
  } else if (statusCode >= 500) {
    return {
      text: 'text-red-400',
      bg: 'bg-red-500/10',
      border: 'border-red-500/20',
      icon: '💥',
      label: 'Server Error'
    };
  } else {
    return {
      text: 'text-purple-400',
      bg: 'bg-purple-500/10',
      border: 'border-purple-500/20',
      icon: '❓',
      label: 'Unknown'
    };
  }
}

/**
 * Get color classes for error levels
 */
export function getErrorLevelColor(level) {
  const normalized = (level || '').toLowerCase();
  switch (normalized) {
    case 'error':
    case 'fatal':
      return {
        text: 'text-red-400',
        bg: 'bg-red-500/10',
        border: 'border-red-500/20',
        dot: 'bg-red-500'
      };
    case 'warning':
      return {
        text: 'text-amber-400',
        bg: 'bg-amber-500/10',
        border: 'border-amber-500/20',
        dot: 'bg-amber-500'
      };
    case 'info':
      return {
        text: 'text-blue-400',
        bg: 'bg-blue-500/10',
        border: 'border-blue-500/20',
        dot: 'bg-blue-500'
      };
    default:
      return {
        text: 'text-slate-400',
        bg: 'bg-slate-500/10',
        border: 'border-slate-500/20',
        dot: 'bg-slate-500'
      };
  }
}

/**
 * Get color classes for issue status
 */
export function getIssueStatusColor(status) {
  const normalized = (status || '').toLowerCase();
  switch (normalized) {
    case 'resolved':
      return {
        text: 'text-emerald-400',
        bg: 'bg-emerald-500/10',
        border: 'border-emerald-500/20',
        icon: '✓'
      };
    case 'ignored':
      return {
        text: 'text-slate-400',
        bg: 'bg-slate-500/10',
        border: 'border-slate-500/20',
        icon: '⊘'
      };
    case 'unresolved':
    default:
      return {
        text: 'text-red-400',
        bg: 'bg-red-500/10',
        border: 'border-red-500/20',
        icon: '●'
      };
  }
}

/**
 * Get color classes for monitor/uptime status
 */
export function getMonitorStatusColor(status) {
  const normalized = (status || '').toLowerCase();
  switch (normalized) {
    case 'up':
      return {
        text: 'text-emerald-400',
        bg: 'bg-emerald-500/10',
        border: 'border-emerald-500/20',
        icon: '✓'
      };
    case 'down':
      return {
        text: 'text-red-400',
        bg: 'bg-red-500/10',
        border: 'border-red-500/20',
        icon: '✕'
      };
    case 'paused':
      return {
        text: 'text-yellow-400',
        bg: 'bg-yellow-500/10',
        border: 'border-yellow-500/20',
        icon: '⏸'
      };
    default:
      return {
        text: 'text-slate-400',
        bg: 'bg-slate-500/10',
        border: 'border-slate-500/20',
        icon: '?'
      };
  }
}

/**
 * Get toast type from HTTP status code
 */
export function getToastTypeFromStatus(statusCode) {
  if (statusCode >= 200 && statusCode < 300) {
    return 'success';
  } else if (statusCode >= 400 && statusCode < 500) {
    if (statusCode === 401 || statusCode === 403) {
      return 'error';
    } else if (statusCode === 404) {
      return 'warning';
    } else if (statusCode === 429) {
      return 'warning';
    } else {
      return 'error';
    }
  } else if (statusCode >= 500) {
    return 'error';
  } else {
    return 'info';
  }
}
