import React from 'react';
import { render, screen } from '@testing-library/react';
import { Card, CardHeader, CardTitle, CardContent } from '../card';
import { Alert, AlertDescription } from '../alert';
import { Badge } from '../badge';
import { Button } from '../button';
import { Input } from '../input';
import { Label } from '../label';
import { Separator } from '../separator';
import { Skeleton } from '../skeleton';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../tabs';
import { Progress } from '../progress';
import { Switch } from '../switch';
import { Textarea } from '../textarea';
import { Checkbox } from '../checkbox';
import { Avatar, AvatarImage, AvatarFallback } from '../avatar';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../table';

// Test simple components that don't have complex dependencies

describe('Simple UI Components', () => {
  describe('Card Components', () => {
    it('should render card components without errors', () => {
      
      render(
        <Card>
          <CardHeader>
            <CardTitle>Test Title</CardTitle>
          </CardHeader>
          <CardContent>Test Content</CardContent>
        </Card>
      );

      expect(screen.getByText('Test Title')).toBeInTheDocument();
      expect(screen.getByText('Test Content')).toBeInTheDocument();
    });
  });

  describe('Alert Components', () => {
    it('should render alert components without errors', () => {
      
      render(
        <Alert>
          <AlertDescription>Test Alert</AlertDescription>
        </Alert>
      );

      expect(screen.getByText('Test Alert')).toBeInTheDocument();
    });
  });

  describe('Badge Component', () => {
    it('should render badge without errors', () => {
      
      render(<Badge>Test Badge</Badge>);

      expect(screen.getByText('Test Badge')).toBeInTheDocument();
    });

    it('should render badge with variant', () => {
      render(<Badge variant="secondary">Secondary Badge</Badge>);

      expect(screen.getByText('Secondary Badge')).toBeInTheDocument();
    });
  });

  describe('Button Component', () => {
    it('should render button without errors', () => {
      render(<Button>Test Button</Button>);

      expect(screen.getByRole('button', { name: 'Test Button' })).toBeInTheDocument();
    });

    it('should render button with variant', () => {
      render(<Button variant="outline">Outline Button</Button>);

      expect(screen.getByRole('button', { name: 'Outline Button' })).toBeInTheDocument();
    });
  });

  describe('Input Component', () => {
    it('should render input without errors', () => {
      render(<Input placeholder="Test input" />);

      expect(screen.getByPlaceholderText('Test input')).toBeInTheDocument();
    });

    it('should render input with type', () => {
      render(<Input type="email" />);

      expect(screen.getByRole('textbox')).toBeInTheDocument();
    });
  });

  describe('Label Component', () => {
    it('should render label without errors', () => {
      render(<Label>Test Label</Label>);

      expect(screen.getByText('Test Label')).toBeInTheDocument();
    });
  });

  describe('Separator Component', () => {
    it('should render separator without errors', () => {
      render(<Separator />);

      // Separator should be in the document
      const separator = document.querySelector('[data-orientation]');
      expect(separator).toBeInTheDocument();
    });
  });

  describe('Skeleton Component', () => {
    it('should render skeleton without errors', () => {
      render(<Skeleton />);

      // Skeleton should have the skeleton class
      const skeleton = document.querySelector('.animate-pulse');
      expect(skeleton).toBeInTheDocument();
    });
  });

  describe('Tabs Components', () => {
    it('should render tabs without errors', () => {
      render(
        <Tabs defaultValue="tab1">
          <TabsList>
            <TabsTrigger value="tab1">Tab 1</TabsTrigger>
            <TabsTrigger value="tab2">Tab 2</TabsTrigger>
          </TabsList>
          <TabsContent value="tab1">Content 1</TabsContent>
          <TabsContent value="tab2">Content 2</TabsContent>
        </Tabs>
      );

      expect(screen.getByText('Tab 1')).toBeInTheDocument();
      expect(screen.getByText('Tab 2')).toBeInTheDocument();
      expect(screen.getByText('Content 1')).toBeInTheDocument();
    });
  });

  describe('Progress Component', () => {
    it('should render progress without errors', () => {
      render(<Progress value={50} />);

      // Progress should be in the document
      const progress = document.querySelector('[role="progressbar"]');
      expect(progress).toBeInTheDocument();
    });
  });

  describe('Switch Component', () => {
    it('should render switch without errors', () => {
      render(<Switch />);

      // Switch should be in the document
      const switchElement = document.querySelector('[role="switch"]');
      expect(switchElement).toBeInTheDocument();
    });
  });

  describe('Textarea Component', () => {
    it('should render textarea without errors', () => {
      render(<Textarea placeholder="Test textarea" />);

      expect(screen.getByPlaceholderText('Test textarea')).toBeInTheDocument();
    });
  });

  describe('Checkbox Component', () => {
    it('should render checkbox without errors', () => {
      render(<Checkbox />);

      // Checkbox should be in the document (it might be a button with role="checkbox")
      const checkbox = document.querySelector('[role="checkbox"]') || document.querySelector('button');
      expect(checkbox).toBeInTheDocument();
    });
  });

  describe('Avatar Components', () => {
    it('should render avatar without errors', () => {
      render(
        <Avatar>
          <AvatarImage src="/test.jpg" alt="Test" />
          <AvatarFallback>TU</AvatarFallback>
        </Avatar>
      );

      expect(screen.getByText('TU')).toBeInTheDocument();
    });
  });

  describe('Table Components', () => {
    it('should render table without errors', () => {
      render(
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Header</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow>
              <TableCell>Cell</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      );

      expect(screen.getByText('Header')).toBeInTheDocument();
      expect(screen.getByText('Cell')).toBeInTheDocument();
    });
  });
});
