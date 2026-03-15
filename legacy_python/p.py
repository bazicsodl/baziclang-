import random
import string

def calculator():
    print("\n--- CALCULATOR ---")
    try:
        expr = input("Enter an expression (e.g., 3 + 5 * 2): ")
        result = eval(expr)
        print("Result:", result)
    except Exception:
        print("Invalid expression!")

def temperature_converter():
    print("\n--- TEMPERATURE CONVERTER ---")
    print("1) Celsius → Fahrenheit")
    print("2) Fahrenheit → Celsius")
    choice = input("Choose 1 or 2: ")

    try:
        if choice == "1":
            c = float(input("Enter °C: "))
            print("Result:", (c * 9/5) + 32, "°F")
        elif choice == "2":
            f = float(input("Enter °F: "))
            print("Result:", (f - 32) * 5/9, "°C")
        else:
            print("Invalid option!")
    except ValueError:
        print("Invalid number!")

def password_generator():
    print("\n--- PASSWORD GENERATOR ---")
    length = input("Enter password length: ")

    if not length.isdigit() or int(length) < 4:
        print("Length must be a number ≥ 4")
        return

    length = int(length)
    characters = string.ascii_letters + string.digits + string.punctuation
    password = "".join(random.choice(characters) for _ in range(length))

    print("Generated password:", password)

def main():
    while True:
        print("\n=== PYTHON MULTI-TOOL ===")
        print("1) Calculator")
        print("2) Temperature Converter")
        print("3) Password Generator")
        print("4) Exit")

        option = input("Choose an option: ")

        if option == "1":
            calculator()
        elif option == "2":
            temperature_converter()
        elif option == "3":
            password_generator()
        elif option == "4":
            print("Goodbye!")
            break
        else:
            print("Invalid choice, try again.")

if __name__ == "__main__":
    main()
