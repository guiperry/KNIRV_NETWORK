// Discourse Topics JavaScript
class DiscourseTopics {
    constructor() {
        this.apiBase = '/.netlify/functions';
        this.auth = new DiscourseAuth();
    }

    async createTopic(topicData) {
        if (!this.auth.isAuthenticated()) {
            throw new Error('Authentication required');
        }

        try {
            const response = await fetch(`${this.apiBase}/discourse-topics`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.auth.getToken()}`
                },
                body: JSON.stringify(topicData)
            });

            return await response.json();
        } catch (error) {
            throw new Error('Failed to create topic');
        }
    }

    async getTopics(params = {}) {
        try {
            const queryString = new URLSearchParams(params).toString();
            const url = `${this.apiBase}/discourse-topics${queryString ? '?' + queryString : ''}`;

            const response = await fetch(url);
            return await response.json();
        } catch (error) {
            throw new Error('Failed to load topics');
        }
    }

    async getTopic(topicId) {
        try {
            const response = await fetch(`${this.apiBase}/discourse-topics/t/${topicId}`);
            return await response.json();
        } catch (error) {
            throw new Error('Failed to load topic');
        }
    }
}

window.DiscourseTopics = DiscourseTopics;