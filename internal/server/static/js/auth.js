// Auth helper functions

function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
    return null;
}

function getAuthToken() {
    // First try localStorage
    let token = localStorage.getItem('auth_token');
    // If not in localStorage, try cookie
    if (!token) {
        token = getCookie('auth_token');
        // If found in cookie, sync to localStorage
        if (token) {
            localStorage.setItem('auth_token', token);
        }
    }
    return token;
}

function setAuthToken(token) {
    localStorage.setItem('auth_token', token);
}

function clearAuthToken() {
    localStorage.removeItem('auth_token');
    // Also clear cookie
    document.cookie = 'auth_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
}

function isAuthenticated() {
    return !!getAuthToken();
}

function redirectToLogin() {
    window.location.href = '/login';
}

// Fetch wrapper with automatic token injection
async function authenticatedFetch(url, options = {}) {
    const token = getAuthToken();
    
    if (!token) {
        redirectToLogin();
        return;
    }
    
    options.headers = {
        ...options.headers,
        'Authorization': `Bearer ${token}`
    };
    
    const response = await fetch(url, options);
    
    // If unauthorized, redirect to login
    if (response.status === 401) {
        clearAuthToken();
        redirectToLogin();
        return;
    }
    
    return response;
}

// Check auth on protected pages
function requireAuth() {
    if (!isAuthenticated()) {
        redirectToLogin();
    }
}

// Logout function
function logout() {
    clearAuthToken();
    // Redirect to logout endpoint to clear server-side cookie properly
    window.location.href = '/logout';
}
