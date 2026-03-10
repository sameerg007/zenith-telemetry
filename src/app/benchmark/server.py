from flask import Flask, request, jsonify
from flask_cors import CORS
import time
import random

app = Flask(__name__)
CORS(app)

class VirtualEquipment:
    def __init__(self, device_id):
        self.device_id = device_id

    def perform_calculation(self):
        # Simulate live calculation
        base_voltage = 3.3
        noise = (random.random() - 0.5) * 0.05
        voltage = base_voltage + noise
        
        resistance = 100.0 + (random.random() - 0.5) * 1.0
        current = voltage / resistance
        
        # Simulate processing delay
        time.sleep(random.randint(1, 5) / 1000.0)
        
        return voltage, current

@app.route('/benchmark', methods=['GET'])
def benchmark():
    try:
        count = int(request.args.get('count', 1))
    except ValueError:
        count = 1

    results = []
    for i in range(count):
        equip = VirtualEquipment(i + 1)
        
        start_time = time.time()
        v, c = equip.perform_calculation()
        end_time = time.time()
        
        latency_ms = (end_time - start_time) * 1000
        
        results.append({
            "device": f"SMU-{i+1}",
            "pyLatency": f"{latency_ms:.3f}",
            "pyData": f"V:{v:.4f},I:{c:.4f}"
        })

    return jsonify(results)

if __name__ == '__main__':
    print("Python Virtual Equipment Server running on port 8000")
    app.run(host='0.0.0.0', port=8000)