import argparse
import sys
import logging
from .daemon import DaemonServer
from .client import run_client
from .config import SHOW_LOGS

def main():
    parser = argparse.ArgumentParser(description="OpenCode Wrapper with Daemon Architecture")
    parser.add_argument("session_name", nargs="?", help="Session name")
    parser.add_argument("message", nargs="?", help="Message, command, or /wait")
    parser.add_argument("--daemon", action="store_true", help="Start the daemon server")
    parser.add_argument("--agent", default="sisyphus", help="Agent name")
    parser.add_argument("--model", default="zai-coding-plan/glm-4.7", help="Model name")
    
    args = parser.parse_args()
    
    if args.daemon:
        DaemonServer().start()
    else:
        if not args.session_name:
            parser.print_help()
            sys.exit(1)
        
        # If message is missing but session provided, maybe user wants status?
        # But for now enforce message or command.
        if not args.message:
            print("Error: Message or command required.")
            sys.exit(1)
            
        run_client(args)

if __name__ == "__main__":
    main()
