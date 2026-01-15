// Discourse Main Application JavaScript
class DiscourseApp {
    constructor() {
        this.currentUser = null;
        this.currentTopic = null;
        this.apiBase = '/.netlify/functions';
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.checkAuthentication();
        this.loadInitialData();
    }

    setupEventListeners() {
        // Navigation
        document.addEventListener('click', (e) => {
            if (e.target.matches('a[href^="/"]')) {
                e.preventDefault();
                this.navigate(e.target.getAttribute('href'));
            }
        });

        // Auth buttons
        const loginBtn = document.getElementById('login-btn');
        const signupBtn = document.getElementById('signup-btn');
        const logoutBtn = document.getElementById('logout-btn');

        if (loginBtn) loginBtn.addEventListener('click', () => this.showLoginModal());
        if (signupBtn) signupBtn.addEventListener('click', () => this.showSignupModal());
        if (logoutBtn) logoutBtn.addEventListener('click', () => this.logout());

        // Create topic button
        const createTopicBtn = document.getElementById('create-topic-btn');
        if (createTopicBtn) {
            createTopicBtn.addEventListener('click', () => this.showComposer());
        }

        // Search
        const searchBtn = document.getElementById('search-btn');
        if (searchBtn) {
            searchBtn.addEventListener('click', () => this.showSearch());
        }
    }

    async checkAuthentication() {
        const token = localStorage.getItem('discourse_token');
        if (token) {
            try {
                const response = await fetch(`${this.apiBase}/discourse-session`, {
                    headers: { 'Authorization': `Bearer ${token}` }
                });

                if (response.ok) {
                    const data = await response.json();
                    this.setCurrentUser(data.user);
                } else {
                    localStorage.removeItem('discourse_token');
                }
            } catch (error) {
                console.error('Auth check failed:', error);
                localStorage.removeItem('discourse_token');
            }
        }
    }

    setCurrentUser(user) {
        this.currentUser = user;
        this.updateUserInterface();
    }

    updateUserInterface() {
        const authButtons = document.getElementById('auth-buttons');
        const userMenu = document.getElementById('user-menu');
        const currentUsername = document.getElementById('current-username');

        if (this.currentUser) {
            if (authButtons) authButtons.style.display = 'none';
            if (userMenu) userMenu.style.display = 'block';
            if (currentUsername) currentUsername.textContent = this.currentUser.username;
        } else {
            if (authButtons) authButtons.style.display = 'block';
            if (userMenu) userMenu.style.display = 'none';
        }
    }

    async loadInitialData() {
        await this.loadCategories();
        await this.loadTopics();
        await this.loadStats();
    }

    async loadCategories() {
        try {
            const response = await fetch(`${this.apiBase}/discourse-categories`);
            const data = await response.json();
            this.renderCategories(data.categories);
        } catch (error) {
            console.error('Failed to load categories:', error);
        }
    }

    async loadTopics() {
        try {
            const response = await fetch(`${this.apiBase}/discourse-topics`);
            const data = await response.json();
            this.renderTopics(data.topics);
        } catch (error) {
            console.error('Failed to load topics:', error);
        }
    }

    renderCategories(categories) {
        const categoriesList = document.getElementById('categories-list');
        if (!categoriesList) return;

        categoriesList.innerHTML = categories.map(category => `
            <li class="category-item">
                <a href="/c/${category.slug}" class="category-link" style="color: ${category.color}">
                    <span class="category-name">${category.name}</span>
                    <span class="topic-count">${category.topic_count || 0}</span>
                </a>
            </li>
        `).join('');
    }

    renderTopics(topics) {
        const topicList = document.getElementById('topic-list');
        if (!topicList) return;

        topicList.innerHTML = topics.map(topic => `
            <div class="topic-item" data-topic-id="${topic.id}">
                <div class="topic-avatar">
                    <img src="${topic.user?.avatar_url || '/img/default-avatar.png'}" alt="${topic.user?.username}">
                </div>
                <div class="topic-details">
                    <h3 class="topic-title">
                        <a href="/t/${topic.slug}/${topic.id}">${topic.title}</a>
                    </h3>
                    <div class="topic-meta">
                        <span class="topic-author">${topic.user?.username}</span>
                        <span class="topic-category">${topic.category?.name || ''}</span>
                        <span class="topic-created">${this.formatDate(topic.created_at)}</span>
                    </div>
                </div>
                <div class="topic-stats">
                    <div class="stat">
                        <span class="stat-number">${topic.reply_count || 0}</span>
                        <span class="stat-label">replies</span>
                    </div>
                    <div class="stat">
                        <span class="stat-number">${topic.views || 0}</span>
                        <span class="stat-label">views</span>
                    </div>
                    <div class="stat">
                        <span class="stat-number">${topic.like_count || 0}</span>
                        <span class="stat-label">likes</span>
                    </div>
                </div>
                <div class="topic-last-post">
                    <span class="last-post-time">${this.formatDate(topic.last_posted_at)}</span>
                </div>
            </div>
        `).join('');
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diff = now - date;

        if (diff < 60000) return 'just now';
        if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
        if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
        if (diff < 2592000000) return `${Math.floor(diff / 86400000)}d ago`;

        return date.toLocaleDateString();
    }

    navigate(path) {
        window.history.pushState({}, '', path);
        this.handleRoute(path);
    }

    handleRoute(path) {
        if (path === '/' || path === '/latest') {
            this.showTopicList();
        } else if (path.startsWith('/t/')) {
            const topicId = path.split('/').pop();
            this.showTopic(topicId);
        } else if (path.startsWith('/u/')) {
            const username = path.split('/').pop();
            this.showUserProfile(username);
        } else if (path.startsWith('/c/')) {
            const categorySlug = path.split('/').pop();
            this.showCategory(categorySlug);
        }
    }

    showTopicList() {
        document.getElementById('topic-list').style.display = 'block';
        document.getElementById('topic-view').style.display = 'none';
    }

    async showTopic(topicId) {
        try {
            const response = await fetch(`${this.apiBase}/discourse-topics/t/${topicId}`);
            const data = await response.json();

            this.currentTopic = data.topic;
            this.renderTopicView(data.topic, data.posts);

            document.getElementById('topic-list').style.display = 'none';
            document.getElementById('topic-view').style.display = 'block';
        } catch (error) {
            console.error('Failed to load topic:', error);
        }
    }

    renderTopicView(topic, posts) {
        const topicView = document.getElementById('topic-view');
        if (!topicView) return;

        topicView.innerHTML = `
            <div class="topic-header">
                <h1 class="topic-title">${topic.title}</h1>
                <div class="topic-meta">
                    <span class="topic-category">${topic.category?.name || ''}</span>
                    <span class="topic-stats">${topic.posts_count} posts, ${topic.views} views</span>
                </div>
            </div>
            <div class="topic-posts">
                ${posts.map((post, index) => this.renderPost(post, index === 0)).join('')}
            </div>
            <div class="topic-actions">
                <button class="btn btn-primary" onclick="DiscourseApp.instance.showReplyComposer()">Reply</button>
            </div>
        `;
    }

    renderPost(post, isFirstPost) {
        return `
            <div class="post" data-post-id="${post.id}">
                <div class="post-avatar">
                    <img src="${post.user?.avatar_url || '/img/default-avatar.png'}" alt="${post.user?.username}">
                </div>
                <div class="post-content">
                    <div class="post-header">
                        <span class="post-username">${post.user?.username}</span>
                        <span class="post-number">#${post.post_number}</span>
                        <span class="post-date">${this.formatDate(post.created_at)}</span>
                    </div>
                    <div class="post-body">
                        ${post.cooked}
                    </div>
                    <div class="post-actions">
                        <button class="post-action like" data-post-id="${post.id}">
                            ❤️ ${post.like_count || 0}
                        </button>
                        <button class="post-action reply" data-post-id="${post.id}">
                            💬 Reply
                        </button>
                    </div>
                </div>
            </div>
        `;
    }

    static initialize() {
        DiscourseApp.instance = new DiscourseApp();
        return DiscourseApp.instance;
    }
}

// Initialize when DOM is ready
if (typeof window !== 'undefined') {
    window.DiscourseApp = DiscourseApp;
}