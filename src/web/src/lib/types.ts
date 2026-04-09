export interface EnergyReading {
  time: string;
  power_w: number;
  solar_w?: number;
  tariff?: string;
}
