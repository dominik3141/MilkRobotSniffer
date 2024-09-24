# Dairy Farm Monitoring System

## Overview

This project is a custom monitoring system for a legacy dairy farm milking setup. It captures and analyzes UDP communication between milking robots and gates to extract valuable statistics about cow movements and milking patterns.

## Features

- Reverse-engineered binary protocol for communication between Texas gates and milking robots
- Packet capture and decoding from a mirrored switch port
- Data storage in SQLite database and Google BigQuery
- Capture and storage of cow images during sorting events
- Analysis of cow movements, gate passages, and milking durations

## Components

1. **UDP Packet Capture**: Uses a Raspberry Pi connected to a managed switch to capture all relevant network traffic.

2. **Packet Decoding**: Decodes the custom binary protocol to extract meaningful data about cow movements and sorting events.

3. **Data Storage**:
   - SQLite database for local storage and querying
   - Google BigQuery for cloud-based storage and advanced analytics

4. **Image Capture**: Takes pictures of cows during sorting events using IP cameras and stores them in Google Cloud Storage.

5. **Data Analysis**: Provides views and queries for analyzing cow movements, milking durations, and gate passage frequencies.

## Setup

1. Connect a Raspberry Pi to the managed switch that links all robots and gates.
2. Configure the switch to mirror all packets to the Raspberry Pi's ethernet port.
3. Install necessary dependencies (Go, SQLite, Google Cloud SDK).
4. Set up Google Cloud credentials for BigQuery and Cloud Storage access.
5. Configure IP camera addresses in the `getPicture.go` file.
6. Run the main program to start capturing and processing data.

## File Structure

- `sqliteBackend.go`: SQLite database setup and operations
- `sorting.go`: Packet decoding and sorting event logic
- `bigQuery.go`: Google BigQuery integration for data storage
- `getPicture.go`: Image capture from IP cameras and storage in Google Cloud Storage

## Usage

1. Start the program to begin capturing and processing network traffic.
2. Query the SQLite database or BigQuery for insights on cow movements and milking patterns.
3. Access stored images in Google Cloud Storage for visual verification of sorting events.

## Notes

- This system is designed for a specific legacy milking setup and may require modifications for use in other environments.
- Ensure proper network security measures are in place when capturing and analyzing network traffic.
- Regularly backup the SQLite database to prevent data loss.

## Future Improvements

- Implement real-time alerting for unusual cow behavior or system anomalies
- Develop a web-based dashboard for easy visualization of farm statistics
- Integrate machine learning models for predictive analytics on cow health and milk production

