import { createCounter, createGauge, createHistogram } from '../index';
describe('Metrics', () => {
    describe('Counter', () => {
        let counter;
        beforeEach(() => {
            counter = createCounter({
                name: 'test_counter',
                help: 'Test counter metric'
            });
        });
        it('should create counter with correct properties', () => {
            expect(counter.getName()).toBe('test_counter');
            expect(counter.getHelp()).toBe('Test counter metric');
            expect(counter.get()).toBe(0);
        });
        it('should increment counter', () => {
            counter.inc();
            expect(counter.get()).toBe(1);
            counter.inc();
            expect(counter.get()).toBe(2);
        });
        it('should add values to counter', () => {
            counter.add(5);
            expect(counter.get()).toBe(5);
            counter.add(3);
            expect(counter.get()).toBe(8);
        });
        it('should handle labeled counters', () => {
            const labels = { method: 'GET', status: '200' };
            counter.inc(labels);
            counter.inc(labels);
            counter.inc();
            expect(counter.get()).toBe(1);
            expect(counter.getWithLabels(labels)).toBe(2);
        });
        it('should reset counter', () => {
            counter.add(10);
            counter.reset();
            expect(counter.get()).toBe(0);
        });
    });
    describe('Gauge', () => {
        let gauge;
        beforeEach(() => {
            gauge = createGauge({
                name: 'test_gauge',
                help: 'Test gauge metric'
            });
        });
        it('should create gauge with correct properties', () => {
            expect(gauge.getName()).toBe('test_gauge');
            expect(gauge.getHelp()).toBe('Test gauge metric');
            expect(gauge.get()).toBe(0);
        });
        it('should set gauge value', () => {
            gauge.set(42);
            expect(gauge.get()).toBe(42);
            gauge.set(0);
            expect(gauge.get()).toBe(0);
        });
        it('should increment and decrement gauge', () => {
            gauge.set(10);
            gauge.inc();
            expect(gauge.get()).toBe(11);
            gauge.dec();
            expect(gauge.get()).toBe(10);
        });
        it('should add and subtract values', () => {
            gauge.set(10);
            gauge.add(5);
            expect(gauge.get()).toBe(15);
            gauge.sub(3);
            expect(gauge.get()).toBe(12);
        });
        it('should handle labeled gauges', () => {
            const labels = { node: 'node1' };
            gauge.set(100, labels);
            gauge.set(200);
            expect(gauge.get()).toBe(200);
            expect(gauge.getWithLabels(labels)).toBe(100);
        });
    });
    describe('Histogram', () => {
        let histogram;
        beforeEach(() => {
            histogram = createHistogram({
                name: 'test_histogram',
                help: 'Test histogram metric',
                buckets: [0.1, 0.5, 1.0, 2.0]
            });
        });
        it('should create histogram with correct properties', () => {
            expect(histogram.getName()).toBe('test_histogram');
            expect(histogram.getHelp()).toBe('Test histogram metric');
            expect(histogram.getBuckets()).toEqual([0.1, 0.5, 1.0, 2.0]);
        });
        it('should observe values', () => {
            histogram.observe(0.05);
            histogram.observe(0.3);
            histogram.observe(0.8);
            histogram.observe(1.5);
            expect(histogram.getCount()).toBe(4);
            expect(histogram.getSum()).toBe(2.65);
        });
        it('should calculate bucket counts correctly', () => {
            histogram.observe(0.05); // <= 0.1
            histogram.observe(0.3); // <= 0.5
            histogram.observe(0.8); // <= 1.0
            histogram.observe(1.5); // <= 2.0
            const bucketCounts = histogram.getBucketCounts();
            expect(bucketCounts).toEqual([1, 2, 3, 4]);
        });
        it('should handle labeled histograms', () => {
            const labels = { endpoint: '/api/users' };
            histogram.observe(0.1);
            histogram.observe(0.2, labels);
            expect(histogram.getCount()).toBe(1);
            expect(histogram.getCount(labels)).toBe(1);
        });
        it('should reset histogram', () => {
            histogram.observe(0.1);
            histogram.observe(0.2);
            histogram.reset();
            expect(histogram.getCount()).toBe(0);
            expect(histogram.getSum()).toBe(0);
        });
    });
    describe('Metric values', () => {
        it('should return structured values for counter', () => {
            const counter = createCounter({
                name: 'test_counter',
                help: 'Test counter',
                labelNames: ['method', 'status']
            });
            counter.inc();
            counter.inc({ method: 'GET', status: '200' });
            const values = counter.getValues();
            expect(values.default).toBe(1);
            expect(values['method="GET",status="200"']).toBe(1);
        });
        it('should return structured values for gauge', () => {
            const gauge = createGauge({
                name: 'test_gauge',
                help: 'Test gauge',
                labelNames: ['node']
            });
            gauge.set(42);
            gauge.set(100, { node: 'node1' });
            const values = gauge.getValues();
            expect(values.default).toBe(42);
            expect(values['node="node1"']).toBe(100);
        });
        it('should return structured values for histogram', () => {
            const histogram = createHistogram({
                name: 'test_histogram',
                help: 'Test histogram',
                buckets: [0.1, 1.0]
            });
            histogram.observe(0.05);
            histogram.observe(0.5);
            const values = histogram.getValues();
            expect(values.buckets).toEqual([0.1, 1.0]);
            expect(values.observations.default.count).toBe(2);
            expect(values.observations.default.sum).toBe(0.55);
            expect(values.observations.default.bucket_counts).toEqual([1, 2]);
        });
    });
});
//# sourceMappingURL=metrics.test.js.map