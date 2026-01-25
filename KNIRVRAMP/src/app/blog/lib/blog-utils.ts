import fs from 'fs';
import path from 'path';

// Define the BlogPost interface with all required properties
export interface BlogPost {
  slug: string;
  category: string;
  date: string;
  title: string;
  subtitle: string;
  author: string;
  readTime: string;
  keywords: string[];
  image: string;
  content?: string;
}

// Function to extract metadata from HTML content
const extractMetadata = (htmlContent: string): Partial<BlogPost> => {
  const metadata: Partial<BlogPost> = {};
  
  // Extract category
  const categoryMatch = htmlContent.match(/<meta name="blog-category" content="([^"]*)">/);
  if (categoryMatch) {
    metadata.category = categoryMatch[1];
  }
  
  // Extract date
  const dateMatch = htmlContent.match(/<meta name="blog-date" content="([^"]*)">/);
  if (dateMatch) {
    metadata.date = dateMatch[1];
  }
  
  // Extract title
  const titleMatch = htmlContent.match(/<meta name="blog-title" content="([^"]*)">/);
  if (titleMatch) {
    metadata.title = titleMatch[1];
  }
  
  // Extract subtitle
  const subtitleMatch = htmlContent.match(/<meta name="blog-subtitle" content="([^"]*)">/);
  if (subtitleMatch) {
    metadata.subtitle = subtitleMatch[1];
  }
  
  // Extract author
  const authorMatch = htmlContent.match(/<meta name="blog-author" content="([^"]*)">/);
  if (authorMatch) {
    metadata.author = authorMatch[1];
  }
  
  // Extract read time
  const readTimeMatch = htmlContent.match(/<meta name="blog-readtime" content="([^"]*)">/);
  if (readTimeMatch) {
    metadata.readTime = readTimeMatch[1];
  }
  
  // Extract keywords
  const keywordsMatch = htmlContent.match(/<meta name="blog-keywords" content="([^"]*)">/);
  if (keywordsMatch) {
    metadata.keywords = keywordsMatch[1].split(',').map(keyword => keyword.trim());
  }
  
  // Extract image
  const imageMatch = htmlContent.match(/<meta name="blog-image" content="([^"]*)">/);
  if (imageMatch) {
    metadata.image = imageMatch[1];
  }
  
  return metadata;
};

// Function to format date for display
export const formatDate = (dateString: string): string => {
  const date = new Date(dateString);
  const options: Intl.DateTimeFormatOptions = { year: 'numeric', month: 'short', day: 'numeric' };
  return date.toLocaleDateString('en-US', options);
};

// Function to get all blog posts from the posts directory
export const getBlogPosts = (): BlogPost[] => {
  const postsDirectory = path.join(process.cwd(), 'src/app/blog/posts');
  const filenames = fs.readdirSync(postsDirectory);
  
  const posts: BlogPost[] = filenames
    .filter(filename => filename.endsWith('.html'))
    .map(filename => {
      const slug = filename.replace('.html', '');
      const filePath = path.join(postsDirectory, filename);
      const htmlContent = fs.readFileSync(filePath, 'utf8');
      
      const metadata = extractMetadata(htmlContent);
      
      return {
        slug,
        ...metadata,
      } as BlogPost;
    })
    .filter(post => post.title) // Filter out invalid posts without title
    .sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime()); // Sort by date (newest first)
  
  return posts;
};

// Function to get a single blog post by slug
export const getBlogPostBySlug = (slug: string): BlogPost | null => {
  const posts = getBlogPosts();
  return posts.find(post => post.slug === slug) || null;
};

// Function to get unique categories
export const getCategories = (): string[] => {
  const posts = getBlogPosts();
  const categories = new Set(posts.map(post => post.category));
  return Array.from(categories);
};