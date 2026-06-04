import { getBlogPostBySlug, formatDate } from "../lib/blog-utils";
import { Calendar, User, Tag, ArrowLeft } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import KnirvLogo from '@/components/KnirvLogo'
import "../styles/blog.css";

interface BlogPostPageProps {
  params: {
    slug: string;
  };
}

export default function BlogPostPage({ params }: BlogPostPageProps) {
  const post = getBlogPostBySlug(params.slug);

  if (!post) {
    return (
      <div className="dve-page flex items-center justify-center">
        <div className="dve-card text-center">
          <h1 className="text-4xl font-bold text-white mb-4">Blog Post Not Found</h1>
          <p className="text-white/70 mb-8">The blog post you're looking for doesn't exist.</p>
          <Link href="/blog">
            <Button className="bg-gradient-to-r from-knirv-primary to-knirv-secondary hover:from-knirv-secondary hover:to-knirv-primary text-white">
              Back to Blog
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="dve-page">
      {/* Navigation */}
      <nav className="dve-nav">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <KnirvLogo />
          </div>
        </div>
      </nav>

      {/* Blog Post Content */}
      <div className="py-12">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <Link href="/blog">
            <Button variant="outline" className="border-knirv-border-primary/50 text-knirv-text-primary hover:bg-knirv-bg-primary/10 mb-8">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Blog
            </Button>
          </Link>

          <article>
            {/* Header */}
            <header className="mb-8">
              <div className="flex items-center space-x-4 mb-4">
                <Badge variant="secondary" className="bg-white/10 text-white/80">
                  {post.category}
                </Badge>
                <div className="flex items-center space-x-2 text-white/60 text-sm">
                  <Calendar className="h-4 w-4" />
                  <span>{formatDate(post.date)}</span>
                </div>
              </div>

              <h1 className="text-4xl md:text-5xl font-bold text-white mb-6 leading-tight">
                {post.title}
              </h1>

              <p className="text-xl text-white/70 mb-6 leading-relaxed">
                {post.subtitle}
              </p>

              <div className="flex items-center justify-between mb-8">
                <div className="flex items-center space-x-2">
                  <User className="h-4 w-4 text-white/60" />
                  <span className="text-white/80 text-sm">{post.author}</span>
                </div>
                <span className="text-white/60 text-sm">{post.readTime}</span>
              </div>

              <div className="flex flex-wrap gap-2 mb-8">
                {post.keywords.map((keyword, index) => (
                  <Badge key={index} variant="outline" className="border-white/20 text-white/60">
                    {keyword}
                  </Badge>
                ))}
              </div>
            </header>

            {/* Featured Image */}
            <div className="bg-gradient-to-br from-knirv-primary/20 to-knirv-secondary/20 p-12 flex items-center justify-center mb-8">
              <span className="text-8xl">{post.image}</span>
            </div>

            {/* Post Content */}
            <div className="blog-content">
              <div dangerouslySetInnerHTML={{ __html: getPostContent(params.slug) }} />
            </div>
          </article>
        </div>
      </div>
    </div>
  );
}

// Function to get the HTML content of a blog post
const getPostContent = (slug: string): string => {
  const fs = require('fs');
  const path = require('path');
  
  const postsDirectory = path.join(process.cwd(), 'src/app/blog/posts');
  const filePath = path.join(postsDirectory, `${slug}.html`);
  
  if (fs.existsSync(filePath)) {
    const content = fs.readFileSync(filePath, 'utf8');
    
    // Extract the body content from the HTML file
    const bodyMatch = content.match(/<body>([\s\S]*)<\/body>/);
    if (bodyMatch) {
      return bodyMatch[1];
    }
  }
  
  return '<p>Content not available</p>';
};
