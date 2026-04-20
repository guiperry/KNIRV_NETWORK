// Discourse Admin JavaScript
class DiscourseAdmin {
    constructor() {
        this.apiBase = '/.netlify/functions';
        this.auth = new DiscourseAuth();
    }

    async getStats() {
        try {
            const response = await fetch(`${this.apiBase}/discourse-admin/stats`, {
                headers: {
                    'Authorization': `Bearer ${this.auth.getToken()}`
                }
            });
            return await response.json();
        } catch (error) {
            throw new Error('Failed to load admin stats');
        }
    }

    async manageUsers() {
        // Implementation for user management
        console.log('User management would be implemented here');
    }

    async manageCategories() {
        // Implementation for category management
        console.log('Category management would be implemented here');
    }
}

window.DiscourseAdmin = DiscourseAdmin;