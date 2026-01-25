import { Bot, Calendar, User, ArrowRight, Tag, TrendingUp, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import Link from "next/link";
import { getBlogPosts, formatDate } from "./lib/blog-utils";

export default function BlogPage() {
  const posts = getBlogPosts();
  const featuredPost = posts[0];
  
  // Calculate category counts
  const categoryCounts = posts.reduce((acc, post) => {
    acc[post.category] = (acc[post.category] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);
  
  const categories = [
    { name: "All Posts", count: posts.length },
    ...Object.entries(categoryCounts).map(([name, count]) => ({ name, count }))
  ];

  return (
  <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-black">
      {/* Navigation */}
  <nav className="border-b border-white/10 bg-slate-900/50 backdrop-blur-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <Link href="/" data-config-nav="home" className="flex items-center space-x-2" aria-label="Go to homepage">
              <Bot className="h-8 w-8 knirv-text-primary" />
              <span className="text-2xl font-bold knirv-gradient-text">
                KNIRV
              </span>
            </Link>
            <div className="flex items-center space-x-4">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-white/40 h-4 w-4" />
                <Input 
                  placeholder="Search articles..." 
                  className="pl-10 bg-white/10 border-white/20 text-white placeholder-white/40 w-64"
                />
              </div>
              <Button variant="outline" className="border-knirv-border-primary/50 text-knirv-text-primary hover:bg-knirv-bg-primary/10">
                Subscribe
              </Button>
            </div>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <div className="py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h1 className="text-6xl md:text-7xl font-bold text-white mb-6">
            <span className="knirv-gradient-text">Blog</span>
          </h1>
          <p className="text-xl text-white/70 mb-8 max-w-3xl mx-auto leading-relaxed">
            Insights, tutorials, and stories from the frontier of AI agent technology.
          </p>
          <div className="flex items-center justify-center space-x-8 text-white/60">
            <div className="flex items-center space-x-2">
              <TrendingUp className="h-5 w-5" />
              <span>Weekly updates</span>
            </div>
            <div className="flex items-center space-x-2">
              <User className="h-5 w-5" />
              <span>Expert authors</span>
            </div>
            <div className="flex items-center space-x-2">
              <Tag className="h-5 w-5" />
              <span>Industry insights</span>
            </div>
          </div>
        </div>
      </div>

      {/* Featured Post */}
      <div className="py-20 bg-black/20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <Badge variant="outline" className="border-knirv-border-primary/50 text-knirv-text-primary mb-4">
              Featured Article
            </Badge>
            <h2 className="text-3xl font-bold text-white">Latest Insights</h2>
          </div>
          
          {featuredPost && (
            <Card className="bg-white/5 border-white/10 backdrop-blur-lg hover:bg-white/10 transition-all duration-300 max-w-4xl mx-auto">
              <CardContent className="p-0">
                <div className="grid md:grid-cols-2 gap-0">
                  <div className="bg-gradient-to-br from-knirv-primary/20 to-knirv-secondary/20 p-12 flex items-center justify-center">
                    <span className="text-8xl">{featuredPost.image}</span>
                  </div>
                  <div className="p-8">
                    <div className="flex items-center space-x-4 mb-4">
                      <Badge variant="secondary" className="bg-white/10 text-white/80">
                        {featuredPost.category}
                      </Badge>
                      <div className="flex items-center space-x-2 text-white/60 text-sm">
                        <Calendar className="h-4 w-4" />
                        <span>{formatDate(featuredPost.date)}</span>
                      </div>
                    </div>
                    
                    <h3 className="text-2xl font-bold text-white mb-4 leading-tight">
                      {featuredPost.title}
                    </h3>
                    
                    <p className="text-white/70 mb-6 leading-relaxed">
                      {featuredPost.subtitle}
                    </p>
                    
                    <div className="flex items-center justify-between mb-6">
                      <div className="flex items-center space-x-2">
                        <User className="h-4 w-4 text-white/60" />
                        <span className="text-white/80 text-sm">{featuredPost.author}</span>
                      </div>
                      <span className="text-white/60 text-sm">{featuredPost.readTime}</span>
                    </div>
                    
                    <div className="flex flex-wrap gap-2 mb-6">
                      {featuredPost.keywords.map((tag, index) => (
                        <Badge key={index} variant="outline" className="border-white/20 text-white/60">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                    
                    <Button className="bg-gradient-to-r from-knirv-primary to-knirv-secondary hover:from-knirv-secondary hover:to-knirv-primary text-white">
                      <Link href={`/blog/${featuredPost.slug}`}>
                        Read Article
                        <ArrowRight className="ml-2 h-4 w-4" />
                      </Link>
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Categories */}
      <div className="py-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-wrap justify-center gap-4">
            {categories.map((category, index) => (
              <Button
                key={index}
                variant={index === 0 ? "default" : "outline"}
                className={index === 0 
                  ? "bg-gradient-to-r from-knirv-primary to-knirv-secondary hover:from-knirv-secondary hover:to-knirv-primary text-white" 
                  : "bg-white/10 border-white/20 text-white/80 hover:bg-white/20 backdrop-blur-lg"
                }
              >
                {category.name} ({category.count})
              </Button>
            ))}
          </div>
        </div>
      </div>

      {/* Blog Posts Grid */}
      <div className="py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <h2 className="text-4xl font-bold text-white text-center mb-16">Recent Articles</h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
            {posts.slice(1).map((post, index) => (
              <Card key={index} className="bg-white/5 border-white/10 backdrop-blur-lg hover:bg-white/10 transition-all duration-300 cursor-pointer group">
                <CardContent className="p-0">
                  <div className="bg-gradient-to-br from-knirv-primary/20 to-knirv-secondary/20 p-8 flex items-center justify-center">
                    <span className="text-4xl">{post.image}</span>
                  </div>
                  
                  <div className="p-6">
                    <div className="flex items-center justify-between mb-3">
                      <Badge variant="secondary" className="bg-white/10 text-white/80">
                        {post.category}
                      </Badge>
                      <div className="flex items-center space-x-2 text-white/60 text-xs">
                        <Calendar className="h-3 w-3" />
                        <span>{formatDate(post.date)}</span>
                      </div>
                    </div>
                    
                    <h3 className="text-lg font-semibold text-white mb-3 group-hover:knirv-text-primary transition-colors leading-tight">
                      {post.title}
                    </h3>
                    
                    <p className="text-white/70 text-sm mb-4 line-clamp-3">
                      {post.subtitle}
                    </p>
                    
                    <div className="flex items-center justify-between text-white/60 text-xs mb-4">
                      <div className="flex items-center space-x-1">
                        <User className="h-3 w-3" />
                        <span>{post.author}</span>
                      </div>
                      <span>{post.readTime}</span>
                    </div>
                    
                    <div className="flex flex-wrap gap-1 mb-4">
                      {post.keywords.map((tag, tagIndex) => (
                        <Badge key={tagIndex} variant="outline" className="border-white/20 text-white/60 text-xs">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                    
                    <Button variant="outline" className="w-full border-knirv-border-primary/50 text-knirv-text-primary hover:bg-knirv-bg-primary/10">
                      <Link href={`/blog/${post.slug}`}>
                        Read More
                        <ArrowRight className="ml-2 h-3 w-3" />
                      </Link>
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </div>

      {/* Newsletter Signup */}
      <div className="py-20 bg-black/20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-4xl font-bold text-white mb-6">Stay Updated</h2>
          <p className="text-xl text-white/70 mb-8 max-w-2xl mx-auto">
            Get the latest insights, tutorials, and industry news delivered to your inbox weekly.
          </p>
          <div className="max-w-md mx-auto flex gap-4">
            <Input 
              placeholder="Enter your email..." 
              className="bg-white/10 border-white/20 text-white placeholder-white/40"
            />
            <Button className="bg-gradient-to-r from-knirv-primary to-knirv-secondary hover:from-knirv-secondary hover:to-knirv-primary text-white">
              Subscribe
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
