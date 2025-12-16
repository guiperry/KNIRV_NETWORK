// Discourse Search JavaScript
class DiscourseSearch {
    constructor() {
        this.apiBase = '/.netlify/functions';
    }

    async search(query, options = {}) {
        try {
            const params = new URLSearchParams({
                q: query,
                ...options
            });

            const response = await fetch(`${this.apiBase}/discourse-search?${params}`);
            return await response.json();
        } catch (error) {
            throw new Error('Search failed');
        }
    }

    showSearchModal() {
        // Implementation for search modal
        console.log('Search modal would be shown here');
    }
}

window.DiscourseSearch = DiscourseSearch;