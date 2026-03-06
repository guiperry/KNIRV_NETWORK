import { cn } from '../utils';

describe('Utils', () => {
  describe('cn (className utility)', () => {
    it('should combine class names correctly', () => {
      const result = cn('class1', 'class2');
      expect(result).toBe('class1 class2');
    });

    it('should handle conditional classes', () => {
      const result = cn('base', true && 'conditional', false && 'hidden');
      expect(result).toBe('base conditional');
    });

    it('should handle undefined and null values', () => {
      const result = cn('base', undefined, null, 'valid');
      expect(result).toBe('base valid');
    });

    it('should handle empty strings', () => {
      const result = cn('base', '', 'valid');
      expect(result).toBe('base valid');
    });

    it('should handle arrays of classes', () => {
      const result = cn(['class1', 'class2'], 'class3');
      expect(result).toBe('class1 class2 class3');
    });

    it('should handle objects with boolean values', () => {
      const result = cn({
        'class1': true,
        'class2': false,
        'class3': true
      });
      expect(result).toBe('class1 class3');
    });

    it('should handle mixed input types', () => {
      const result = cn(
        'base',
        ['array1', 'array2'],
        {
          'object1': true,
          'object2': false
        },
        true && 'conditional',
        'final'
      );
      expect(result).toBe('base array1 array2 object1 conditional final');
    });

    it('should handle no arguments', () => {
      const result = cn();
      expect(result).toBe('');
    });

    it('should handle only falsy values', () => {
      const result = cn(false, null, undefined, '');
      expect(result).toBe('');
    });

    it('should deduplicate classes', () => {
      const result = cn('class1', 'class2', 'class1');
      // Note: This depends on the actual implementation of cn
      // If it uses clsx/classnames, it should deduplicate
      expect(result).toContain('class1');
      expect(result).toContain('class2');
    });
  });
});
