from datetime import datetime


def get_current_time(format: str = "short", **kwargs):
    """
    Simple handler for the 'get_current_time' tool.

    Args:
        format (str): The format of the time to return ('short').

    Returns:
        str: The current time formatted as 'HH:MM'.
    """

    # if format != "short":
    #     raise ValueError("Invalid format. Only 'short' is supported.")
    print(kwargs)
    print("----------", format)

    # Get the current time
    current_time = datetime.now()

    # Format the time as 'HH:MM'
    return current_time.strftime("%H:%M")
