// Discourse Composer JavaScript
class DiscourseComposer {
    constructor() {
        this.apiBase = '/.netlify/functions';
        this.auth = new DiscourseAuth();
        this.isVisible = false;
    }

    show(options = {}) {
        const composer = document.getElementById('composer');
        if (!composer) return;

        this.isVisible = true;
        composer.style.display = 'block';
        composer.innerHTML = this.renderComposer(options);
        this.setupComposerEvents();
    }

    hide() {
        const composer = document.getElementById('composer');
        if (composer) {
            composer.style.display = 'none';
            this.isVisible = false;
        }
    }

    renderComposer(options) {
        const isReply = options.topicId;
        const title = isReply ? 'Reply' : 'Create Topic';

        return `
            <div class="composer-container">
                <div class="composer-header">
                    <h3>${title}</h3>
                    <button class="composer-close" onclick="DiscourseComposer.instance.hide()">×</button>
                </div>
                <form class="composer-form" id="composer-form">
                    ${!isReply ? `
                        <div class="form-group">
                            <input type="text" id="topic-title" placeholder="Topic title" required>
                        </div>
                        <div class="form-group">
                            <select id="topic-category">
                                <option value="">Select category</option>
                            </select>
                        </div>
                    ` : ''}
                    <div class="form-group">
                        <textarea id="topic-content" placeholder="What's on your mind?" required></textarea>
                    </div>
                    <div class="composer-actions">
                        <button type="submit" class="btn btn-primary">
                            ${isReply ? 'Reply' : 'Create Topic'}
                        </button>
                        <button type="button" class="btn btn-secondary" onclick="DiscourseComposer.instance.hide()">
                            Cancel
                        </button>
                    </div>
                </form>
            </div>
        `;
    }

    setupComposerEvents() {
        const form = document.getElementById('composer-form');
        if (form) {
            form.addEventListener('submit', (e) => {
                e.preventDefault();
                this.submitComposer();
            });
        }
    }

    async submitComposer() {
        if (!this.auth.isAuthenticated()) {
            alert('Please log in to post');
            return;
        }

        const title = document.getElementById('topic-title')?.value;
        const content = document.getElementById('topic-content')?.value;
        const category = document.getElementById('topic-category')?.value;

        if (!content) {
            alert('Please enter some content');
            return;
        }

        try {
            const topicData = {
                title,
                raw: content,
                category_id: category || null
            };

            const response = await fetch(`${this.apiBase}/discourse-topics`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.auth.getToken()}`
                },
                body: JSON.stringify(topicData)
            });

            const result = await response.json();

            if (result.topic) {
                this.hide();
                window.location.href = `/t/${result.topic.slug}/${result.topic.id}`;
            } else {
                alert('Failed to create topic');
            }
        } catch (error) {
            alert('Error creating topic');
        }
    }

    static initialize() {
        DiscourseComposer.instance = new DiscourseComposer();
        return DiscourseComposer.instance;
    }
}

window.DiscourseComposer = DiscourseComposer;