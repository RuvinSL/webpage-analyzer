   async function analyzeURL() {
            const urlInput = document.getElementById('url');
            const url = urlInput.value.trim();
            
            if (!url) {
                showError('Please enter a valid URL');
                return;
            }

            // URL validation
            try {
                new URL(url);
            } catch (e) {
                console.error('URL validation error:', e);
                showError('Please enter a valid URL starting with http:// or https://');
                return;
            }

            // UI state
            const btn = document.getElementById('analyzeBtn');
            const loader = document.getElementById('loader');
            const results = document.getElementById('results');
            const error = document.getElementById('error');
            
            btn.disabled = true;
            loader.style.display = 'block';
            results.style.display = 'none';
            error.style.display = 'none';

            try {
                const response = await fetch('/api/v1/analyze', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ url: url })
                });

                const data = await response.json();

                if (!response.ok) {
                    throw new Error(data.error || 'Analysis failed');
                }

                displayResults(data);
            } catch (err) {
                showError(err.message || 'Failed to analyze URL. Please try again.');
            } finally {
                btn.disabled = false;
                loader.style.display = 'none';
            }
        }

        function displayResults(data) {
            document.getElementById('results').style.display = 'block';
            
            // Document info
            document.getElementById('htmlVersion').textContent = data.html_version || 'Unknown';
            document.getElementById('pageTitle').textContent = data.title || 'No title';
            document.getElementById('loginForm').innerHTML = data.has_login_form 
                ? '<span class="badge badge-success">Found</span>' 
                : '<span class="badge badge-info">Not Found</span>';

            // Headings
            const headingsList = document.getElementById('headingsList');
            headingsList.innerHTML = '';
            
            for (let level = 1; level <= 6; level++) {
                const count = data.headings[`h${level}`] || 0;
                if (count > 0) {
                    const item = document.createElement('div');
                    item.className = 'heading-item';
                    item.innerHTML = `<span class="heading-level">H${level}</span>: ${count}`;
                    headingsList.appendChild(item);
                }
            }

            if (headingsList.children.length === 0) {
                headingsList.innerHTML = '<span style="color: #7f8c8d;">No headings found</span>';
            }

            // Links
            document.getElementById('totalLinks').textContent = data.links.total || 0;
            document.getElementById('internalLinks').textContent = data.links.internal || 0;
            document.getElementById('externalLinks').textContent = data.links.external || 0;
            document.getElementById('inaccessibleLinks').textContent = data.links.inaccessible || 0;
        }

        function showError(message) {
            const error = document.getElementById('error');
            const errorMessage = document.getElementById('errorMessage');
            
            errorMessage.textContent = message;
            error.style.display = 'block';
        }

        // Enter key support
        document.getElementById('url').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                analyzeURL();
            }
        });