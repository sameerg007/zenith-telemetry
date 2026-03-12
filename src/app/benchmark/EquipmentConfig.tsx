"use client";
import React, { useState } from "react";

interface EquipmentConfigProps {
  onChange: (count: number) => void;
  min?: number;
  max?: number;
}

export default function EquipmentConfig({ onChange, min = 1, max = 50 }: EquipmentConfigProps) {
  const [numEquipments, setNumEquipments] = useState(min);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = parseInt(e.target.value, 10);
    if (!isNaN(value) && value >= min && value <= max) {
      setNumEquipments(value);
      onChange(value);
    }
  };

  return (
    <div className="mb-8">
      <label htmlFor="num-equipments" className="font-medium mr-2">
        Number of Equipments:
      </label>
      <input
        id="num-equipments"
        type="number"
        min={min}
        max={max}
        value={numEquipments}
        onChange={handleChange}
        className="border px-2 py-1 rounded w-24"
      />
    </div>
  );
}
