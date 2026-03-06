import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '../form';
import { Input } from '../input';
import { Button } from '../button';

// Test schema
const testSchema = z.object({
  email: z.string().email('Invalid email address'),
  name: z.string().min(2, 'Name must be at least 2 characters'),
});

type TestFormData = z.infer<typeof testSchema>;

// Test form component
const TestForm: React.FC<{
  onSubmit: (data: TestFormData) => void;
  defaultValues?: Partial<TestFormData>;
}> = ({ onSubmit, defaultValues }) => {
  const form = useForm<TestFormData>({
    resolver: zodResolver(testSchema),
    defaultValues: defaultValues || {
      email: '',
      name: '',
    },
  });

  const handleSubmit = (data: TestFormData) => {
    onSubmit(data);
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSubmit)} data-testid="test-form">
        <FormField
          control={form.control}
          name="email"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Email</FormLabel>
              <FormControl>
                <Input placeholder="Enter email" {...field} />
              </FormControl>
              <FormDescription>
                Your email address
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input placeholder="Enter name" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        
        <Button type="submit" data-testid="submit-button">
          Submit
        </Button>
      </form>
    </Form>
  );
};

describe('Form Components', () => {
  const mockSubmit = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render form with all fields', () => {
    render(<TestForm onSubmit={mockSubmit} />);

    expect(screen.getByLabelText('Email')).toBeInTheDocument();
    expect(screen.getByLabelText('Name')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter email')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter name')).toBeInTheDocument();
    expect(screen.getByText('Your email address')).toBeInTheDocument();
    expect(screen.getByTestId('submit-button')).toBeInTheDocument();
  });

  it('should display validation errors for invalid input', async () => {
    render(<TestForm onSubmit={mockSubmit} />);

    const submitButton = screen.getByTestId('submit-button');
    
    // Submit form with empty fields
    fireEvent.click(submitButton);

    // Wait for validation errors to appear
    await screen.findByText('Invalid email address');
    expect(screen.getByText('Name must be at least 2 characters')).toBeInTheDocument();
    
    // Form should not be submitted
    expect(mockSubmit).not.toHaveBeenCalled();
  });

  it('should submit form with valid data', async () => {
    render(<TestForm onSubmit={mockSubmit} />);

    const emailInput = screen.getByPlaceholderText('Enter email');
    const nameInput = screen.getByPlaceholderText('Enter name');
    const submitButton = screen.getByTestId('submit-button');

    // Fill in valid data
    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    fireEvent.change(nameInput, { target: { value: 'John Doe' } });

    // Submit form
    fireEvent.click(submitButton);

    // Wait for form submission
    await new Promise(resolve => setTimeout(resolve, 100));

    expect(mockSubmit).toHaveBeenCalledWith({
      email: 'test@example.com',
      name: 'John Doe',
    });
  });

  it('should populate form with default values', () => {
    const defaultValues = {
      email: 'default@example.com',
      name: 'Default Name',
    };

    render(<TestForm onSubmit={mockSubmit} defaultValues={defaultValues} />);

    const emailInput = screen.getByDisplayValue('default@example.com');
    const nameInput = screen.getByDisplayValue('Default Name');

    expect(emailInput).toBeInTheDocument();
    expect(nameInput).toBeInTheDocument();
  });

  it('should clear validation errors when input becomes valid', async () => {
    render(<TestForm onSubmit={mockSubmit} />);

    const emailInput = screen.getByPlaceholderText('Enter email');
    const submitButton = screen.getByTestId('submit-button');

    // Submit to trigger validation errors
    fireEvent.click(submitButton);
    await screen.findByText('Invalid email address');

    // Fix the email
    fireEvent.change(emailInput, { target: { value: 'valid@example.com' } });
    
    // Trigger validation by blurring
    fireEvent.blur(emailInput);

    // Wait for error to clear
    await new Promise(resolve => setTimeout(resolve, 100));

    expect(screen.queryByText('Invalid email address')).not.toBeInTheDocument();
  });

  it('should handle form reset', () => {
    const TestFormWithReset: React.FC = () => {
      const form = useForm<TestFormData>({
        resolver: zodResolver(testSchema),
        defaultValues: { email: '', name: '' },
      });

      return (
        <Form {...form}>
          <form data-testid="test-form">
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Input placeholder="Enter email" {...field} />
                  </FormControl>
                </FormItem>
              )}
            />
            <Button 
              type="button" 
              onClick={() => form.reset()}
              data-testid="reset-button"
            >
              Reset
            </Button>
          </form>
        </Form>
      );
    };

    render(<TestFormWithReset />);

    const emailInput = screen.getByPlaceholderText('Enter email');
    const resetButton = screen.getByTestId('reset-button');

    // Enter some data
    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    expect(emailInput).toHaveValue('test@example.com');

    // Reset form
    fireEvent.click(resetButton);
    expect(emailInput).toHaveValue('');
  });
});
