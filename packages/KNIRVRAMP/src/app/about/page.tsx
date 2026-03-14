'use client';

import React from 'react';
import { Network, Brain, Zap, Shield, Globe, Users, Target, Lightbulb, Linkedin } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import Link from 'next/link';
import Image from 'next/image';
import KnirvNetworkLogo from '../../components/KnirvNetworkLogo.jsx';
import KnirvLogo from '@/components/KnirvLogo';

export default function About() {
  const features = [
    {
      icon: Brain,
      title: 'Neural Intelligence',
      description: 'Advanced AI models optimized for decentralized execution and edge deployment.'
    },
    {
      icon: Network,
      title: 'Decentralized Network',
      description: 'Distributed architecture ensuring resilience, scalability, and global accessibility.'
    },
    {
      icon: Shield,
      title: 'Trusted Execution',
      description: 'Secure environments with cryptographic verification and privacy protection.'
    },
    {
      icon: Zap,
      title: 'High Performance',
      description: 'Optimized for speed and efficiency across mobile, desktop, and cloud platforms.'
    },
    {
      icon: Globe,
      title: 'Cross-Platform',
      description: 'Deploy anywhere - from mobile apps to desktop software to cloud infrastructure.'
    },
    {
      icon: Users,
      title: 'Developer Friendly',
      description: 'Intuitive tools and comprehensive APIs for seamless integration and development.'
    }
  ];

  const stats = [
    { label: 'Active Nodes', value: '10,000+' },
    { label: 'Models Deployed', value: '50,000+' },
    { label: 'Developers', value: '5,000+' },
    { label: 'Countries', value: '120+' }
  ];

  const teamMembers = [
    {
      name: 'Guillermo Perry',
      role: 'Chief Solutions Architect',
      image: '/images/team/JohnPerry.png',
      linkedin: 'https://www.linkedin.com/in/guillermo-perry-7b29aa30/',
      description: 'Guillermo Perry is a seasoned technology leader with over 20 years of experience in software architecture and distributed systems. As Chief Solutions Architect, he leads the technical vision for KNIRV, focusing on scalable infrastructure and innovative AI deployment strategies.',
      skills: [
        { name: 'System Architecture', level: 95 },
        { name: 'Distributed Systems', level: 90 },
        { name: 'AI/ML Infrastructure', level: 85 },
        { name: 'Cloud Computing', level: 88 }
      ]
    },
    {
      name: 'Herman Duquerronette',
      role: 'Sr. Software Developer',
      image: '/images/team/Herman.jpg',
      linkedin: 'https://www.linkedin.com/in/herman-duquerronette/',
      description: 'A seasoned software developer with over two decades of experience in the industry. His passion for technology is matched only by his commitment to excellence. Herman brings deep technical expertise to the KNIRV ecosystem, focusing on robust software development practices and system reliability across the D-TEN infrastructure.',
      skills: [
        { name: 'Blockchain Development', level: 92 },
        { name: 'Cryptography', level: 88 },
        { name: 'Rust/Go', level: 90 },
        { name: 'Smart Contracts', level: 85 }
      ]
    },
    {
      name: 'Kareem Bullard',
      role: 'Project Manager',
      image: '/images/team/Kareem.jpg',
      linkedin: 'https://www.linkedin.com/in/kareem-bullard-a644b9225',
      description: 'Orchestrating Success with Strategic Leadership. With a proven track record in delivering high-impact projects on time and within budget, I bring a unique blend of tactical foresight and operational expertise. Kareem ensures seamless coordination across all KNIRV D-TEN development initiatives, managing complex project timelines and stakeholder relationships to drive successful outcomes.',
      skills: [
        { name: 'Agile/Scrum', level: 93 },
        { name: 'Product Strategy', level: 87 },
        { name: 'Team Leadership', level: 90 },
        { name: 'Stakeholder Management', level: 89 }
      ]
    }
  ];



  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-black">
      {/* Navigation */}
      <nav className="border-b border-white/10 bg-slate-900/50 backdrop-blur-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <KnirvLogo />
            <Link href="/" className="text-white/70 hover:text-white transition-colors">
              ← Back to Home
            </Link>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="py-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <div className="flex justify-center mb-8">
            <div className="w-[400px] h-[200px] bg-gradient-to-br from-slate-900 via-slate-800 to-black rounded-2xl p-4 shadow-2xl knirv-glow">
              <KnirvNetworkLogo />
            </div>
          </div>

          <h1 className="text-5xl md:text-6xl font-bold text-white mb-6">
            About <span className="knirv-gradient-text">KNIRV</span>
          </h1>
          <p className="text-xl text-white/70 mb-12 max-w-3xl mx-auto leading-relaxed">
            KNIRV (Key Neural Intelligence Reasoning Validation) is a revolutionary
            decentralized platform for neural intelligence deployment. We're building the future of
            AI where models run everywhere, securely and efficiently.
          </p>

          {/* Stats */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mb-16">
            {stats.map((stat, index) => (
              <div key={index} className="text-center">
                <div className="text-3xl font-bold knirv-gradient-text mb-2">{stat.value}</div>
                <div className="text-white/70">{stat.label}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Mission Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto">
          <Card className="knirv-card-gradient backdrop-blur-lg">
            <CardHeader className="text-center">
              <CardTitle className="text-3xl text-white mb-4">Our Mission</CardTitle>
            </CardHeader>
            <CardContent className="text-center">
              <p className="text-lg text-white/80 max-w-4xl mx-auto leading-relaxed">
                To enable trusted artificial intelligence by creating a decentralized trust execution network where
                neural models can be deployed, trained, tested, and verified across any platform. 
                
                We believe that every device should have access to trusted AI decision-making capabilities, without compromising security or privacy.

                Our mission is simple and our name says it all:
                
                **K**ey **N**eural **I**ntelligence **R**easoning **V**alidation
              </p>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* Values Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-6xl mx-auto">
          <h2 className="text-3xl font-bold text-white text-center mb-12">
            Why Choose <span className="knirv-gradient-text">KNIRV</span>?
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            {features.map((feature, index) => {
              const IconComponent = feature.icon;
              return (
                <Card key={index} className="knirv-card-gradient backdrop-blur-lg">
                  <CardHeader>
                    <IconComponent className="h-12 w-12 knirv-text-primary mb-4" />
                    <CardTitle className="text-white">{feature.title}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-white/70">{feature.description}</p>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </div>
      </section>

      {/* Technology Stack */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-6xl mx-auto">
          <Card className="knirv-card-gradient backdrop-blur-lg">
            <CardHeader className="text-center">
              <CardTitle className="text-3xl text-white mb-4">Technology Stack</CardTitle>
              <CardDescription className="text-white/70 text-lg">
                Built on cutting-edge technologies for maximum performance and reliability
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {[
                  'WebAssembly', 'Rust', 'TypeScript', 'Go',
                  'PyTorch', 'TensorFlow', 'ONNX', 'Blockchain',
                  'Docker', 'Kubernetes', 'AWS', 'CloudFlare'
                ].map((tech, index) => (
                  <Badge key={index} variant="outline" className="knirv-border-primary knirv-text-primary text-center py-2">
                    {tech}
                  </Badge>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* Team Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-6xl mx-auto">
          <h2 className="text-3xl font-bold text-white text-center mb-4">
            Meet Our <span className="knirv-gradient-text">Team</span>
          </h2>
          <p className="text-white/70 text-center mb-12 max-w-2xl mx-auto">
            Our leadership team brings decades of combined experience in distributed systems, 
            AI infrastructure, and blockchain technology.
          </p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            {teamMembers.map((member, index) => (
              <Card key={index} className="knirv-card-gradient backdrop-blur-lg overflow-hidden">
                <CardContent className="p-6">
                  <div className="flex flex-col items-center text-center">
                    {/* Team Member Image */}
                    <div className="relative w-32 h-32 mb-4 rounded-full overflow-hidden border-4 border-cyan-500/30">
                      <Image
                        src={member.image}
                        alt={member.name}
                        fill
                        className="object-cover"
                      />
                    </div>
                    
                    {/* Name and Role */}
                    <h3 className="text-xl font-bold text-white mb-1">{member.name}</h3>
                    <p className="text-cyan-400 text-sm mb-3">{member.role}</p>
                    
                    {/* LinkedIn Link */}
                    <a 
                      href={member.linkedin}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2 text-white/70 hover:text-cyan-400 transition-colors mb-4"
                    >
                      <Linkedin className="h-4 w-4" />
                      <span className="text-sm">Connect on LinkedIn</span>
                    </a>
                    
                    {/* Description */}
                    <p className="text-white/70 text-sm leading-relaxed mb-4">
                      {member.description}
                    </p>
                    
                    {/* Skills */}
                    <div className="w-full space-y-3 mt-4">
                      <h4 className="text-white font-semibold text-sm mb-2">Key Skills</h4>
                      {member.skills.map((skill, skillIndex) => (
                        <div key={skillIndex} className="space-y-1">
                          <div className="flex justify-between text-xs text-white/70">
                            <span>{skill.name}</span>
                            <span>{skill.level}%</span>
                          </div>
                          <div className="w-full bg-slate-700/50 rounded-full h-2">
                            <div 
                              className="knirv-gradient h-2 rounded-full transition-all duration-500"
                              style={{ width: `${skill.level}%` }}
                            />
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <Card className="knirv-card-gradient backdrop-blur-lg">
            <CardHeader>
              <CardTitle className="text-3xl text-white mb-4">The Future is Decentralized</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-lg text-white/80 leading-relaxed mb-6">
                We envision a world where AI models run seamlessly across billions of devices,
                from smartphones to IoT sensors, all connected through the KNIRV network.
                A future where intelligence is distributed, privacy is preserved, and innovation
                knows no boundaries.
              </p>
              <Link href="/" className="inline-block">
                <button className="knirv-gradient hover:opacity-90 px-8 py-3 rounded-lg text-white font-semibold transition-opacity">
                  Start Building with KNIRV
                </button>
              </Link>
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  );
}
