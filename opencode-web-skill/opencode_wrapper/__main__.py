import argparse
import sys
import logging
from .daemon import DaemonServer
from .client import run_client
from .config import SHOW_LOGS

def main():
    parser = argparse.ArgumentParser(description="OpenCode Wrapper with Daemon Architecture")
    parser.add_argument("session_name", nargs="?", help="Session name")
    parser.add_argument("message", nargs="*", help="Message, command, or /wait")
    parser.add_argument("--daemon", action="store_true", help="Start the daemon server")
    parser.add_argument("--restart", action="store_true", help="Restart the daemon server")
    parser.add_argument("--agent", default="sisyphus", help="Agent name")
    parser.add_argument("--model", default="zai-coding-plan/glm-5", help="Model name")
    
    args = parser.parse_args()
    
    if args.restart:
        from .config import PID_FILE
        import os
        import signal
        import time
        if PID_FILE.exists():
            try:
                pid = int(PID_FILE.read_text().strip())
                print(f"Stopping daemon (PID {pid})...")
                os.kill(pid, signal.SIGTERM)
                # Wait for it to stop
                for _ in range(10):
                    try:
                        os.kill(pid, 0)
                        time.sleep(0.5)
                    except ProcessLookupError:
                        break
                if PID_FILE.exists():
                    PID_FILE.unlink()
            except Exception as e:
                print(f"Error stopping daemon: {e}")
        
        print("Starting daemon...")
        DaemonServer().start()
    elif args.daemon:
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
