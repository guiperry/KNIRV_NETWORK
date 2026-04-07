/**
 * In-memory Prometheus-like metrics implementation
 * This provides a compatible interface without requiring the full Prometheus client
 */
export class Metric {
    constructor(options) {
        this.name = options.name;
        this.help = options.help;
        this.labelNames = options.labelNames || [];
    }
    getName() {
        return this.name;
    }
    getHelp() {
        return this.help;
    }
    getLabelNames() {
        return this.labelNames;
    }
}
export class Counter extends Metric {
    constructor(options) {
        super(options);
        this.value = 0;
        this.labeledValues = new Map();
    }
    inc(labels) {
        if (labels) {
            const key = this.getLabelsKey(labels);
            this.labeledValues.set(key, (this.labeledValues.get(key) || 0) + 1);
        }
        else {
            this.value++;
        }
    }
    add(value, labels) {
        if (labels) {
            const key = this.getLabelsKey(labels);
            this.labeledValues.set(key, (this.labeledValues.get(key) || 0) + value);
        }
        else {
            this.value += value;
        }
    }
    get() {
        return this.value;
    }
    getWithLabels(labels) {
        const key = this.getLabelsKey(labels);
        return this.labeledValues.get(key) || 0;
    }
    getTotal() {
        let total = this.value;
        this.labeledValues.forEach(v => total += v);
        return total;
    }
    reset() {
        this.value = 0;
        this.labeledValues.clear();
    }
    getLabelsKey(labels) {
        return this.labelNames
            .map(name => `${name}="${labels[name] || ''}"`)
            .join(',');
    }
    getValues() {
        const values = {};
        if (this.value > 0) {
            values.default = this.value;
        }
        this.labeledValues.forEach((value, key) => {
            values[key] = value;
        });
        return values;
    }
}
export class Gauge extends Metric {
    constructor(options) {
        super(options);
        this.value = 0;
        this.labeledValues = new Map();
    }
    set(value, labels) {
        if (labels) {
            const key = this.getLabelsKey(labels);
            this.labeledValues.set(key, value);
        }
        else {
            this.value = value;
        }
    }
    inc(labels) {
        if (labels) {
            const key = this.getLabelsKey(labels);
            this.labeledValues.set(key, (this.labeledValues.get(key) || 0) + 1);
        }
        else {
            this.value++;
        }
    }
    dec(labels) {
        if (labels) {
            const key = this.getLabelsKey(labels);
            this.labeledValues.set(key, (this.labeledValues.get(key) || 0) - 1);
        }
        else {
            this.value--;
        }
    }
    add(value, labels) {
        if (labels) {
            const key = this.getLabelsKey(labels);
            this.labeledValues.set(key, (this.labeledValues.get(key) || 0) + value);
        }
        else {
            this.value += value;
        }
    }
    sub(value, labels) {
        if (labels) {
            const key = this.getLabelsKey(labels);
            this.labeledValues.set(key, (this.labeledValues.get(key) || 0) - value);
        }
        else {
            this.value -= value;
        }
    }
    get() {
        return this.value;
    }
    getWithLabels(labels) {
        const key = this.getLabelsKey(labels);
        return this.labeledValues.get(key) || 0;
    }
    reset() {
        this.value = 0;
        this.labeledValues.clear();
    }
    getLabelsKey(labels) {
        return this.labelNames
            .map(name => `${name}="${labels[name] || ''}"`)
            .join(',');
    }
    getValues() {
        const values = {};
        values.default = this.value;
        this.labeledValues.forEach((value, key) => {
            values[key] = value;
        });
        return values;
    }
}
export class Histogram extends Metric {
    constructor(options) {
        super(options);
        this.observations = [];
        this.labeledObservations = new Map();
        this.buckets = options.buckets || [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10];
    }
    observe(value, labels) {
        if (labels) {
            const key = this.getLabelsKey(labels);
            const observations = this.labeledObservations.get(key) || [];
            observations.push(value);
            this.labeledObservations.set(key, observations);
        }
        else {
            this.observations.push(value);
        }
    }
    getBuckets() {
        return [...this.buckets];
    }
    getBucketCounts(labels) {
        const observations = labels
            ? this.labeledObservations.get(this.getLabelsKey(labels)) || []
            : this.observations;
        return this.buckets.map(bucket => observations.filter(value => value <= bucket).length);
    }
    getCount(labels) {
        const observations = labels
            ? this.labeledObservations.get(this.getLabelsKey(labels)) || []
            : this.observations;
        return observations.length;
    }
    getSum(labels) {
        const observations = labels
            ? this.labeledObservations.get(this.getLabelsKey(labels)) || []
            : this.observations;
        return observations.reduce((sum, value) => sum + value, 0);
    }
    getLabelsKey(labels) {
        return this.labelNames
            .map(name => `${name}="${labels[name] || ''}"`)
            .join(',');
    }
    getValues() {
        const result = {
            buckets: this.buckets,
            observations: {
                default: {
                    count: this.getCount(),
                    sum: this.getSum(),
                    bucket_counts: this.getBucketCounts()
                }
            }
        };
        this.labeledObservations.forEach((observations, key) => {
            result.observations[key] = {
                count: observations.length,
                sum: observations.reduce((sum, value) => sum + value, 0),
                bucket_counts: this.getBucketCounts(Object.fromEntries(key.split(',').map(pair => {
                    const [name, value] = pair.split('=');
                    return [name, value.replace(/"/g, '')];
                })))
            };
        });
        return result;
    }
    reset() {
        this.observations = [];
        this.labeledObservations.clear();
    }
}
//# sourceMappingURL=metrics.js.map