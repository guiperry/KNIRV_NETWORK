// Discourse Authentication JavaScript
class DiscourseAuth {
    constructor() {
        this.apiBase = '/.netlify/functions';
    }

    async login(credentials) {
        try {
            const response = await fetch(`${this.apiBase}/discourse-session`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(credentials)
            });

            const data = await response.json();

            if (data.success) {
                localStorage.setItem('discourse_token', data.token);
                return { success: true, user: data.user };
            } else {
                return { success: false, error: data.error };
            }
        } catch (error) {
            return { success: false, error: 'Network error' };
        }
    }

    async signup(userData) {
        try {
            const response = await fetch(`${this.apiBase}/discourse-users`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(userData)
            });

            const data = await response.json();
            return data;
        } catch (error) {
            return { success: false, error: 'Network error' };
        }
    }

    logout() {
        localStorage.removeItem('discourse_token');
        window.location.reload();
    }

    getToken() {
        return localStorage.getItem('discourse_token');
    }

    isAuthenticated() {
        return !!this.getToken();
    }
}

window.DiscourseAuth = DiscourseAuth;